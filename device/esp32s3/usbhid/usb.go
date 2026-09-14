// Package usbhid exposes two buttons or keyboard keys through ESP32-S3 USB OTG.
// It owns the single internal USB PHY while started, replacing USB Serial/JTAG.
package usbhid

import (
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/mmio"
)

// ESP32-S3 DWC2 full-speed device registers. FIFO sizes/addresses are in
// 32-bit words; endpoint register blocks have a 0x20-byte stride.
// Register definitions: Espressif ESP-IDF soc/esp32s3 and Synopsys DWC2.
const base = uintptr(0x60080000)
const (
	gotgctl   = 0x0
	gotgint   = 0x4
	gahbcfg   = 0x8
	gusbcfg   = 0xc
	grstctl   = 0x10
	gintsts   = 0x14
	gintmsk   = 0x18
	grxstsp   = 0x20
	grxfsiz   = 0x24
	dieptxf0  = 0x28
	gdfifocfg = 0x5c
	dieptxf1  = 0x104
	dcfg      = 0x800
	dctl      = 0x804
	dsts      = 0x808
	diepmsk   = 0x810
	doepmsk   = 0x814
	daintmsk  = 0x81c
	diepctl0  = 0x900
	diepint0  = 0x908
	diepctl1  = 0x920
	diepint1  = 0x928
	doepctl0  = 0xb00
	doepint0  = 0xb08
	doeptsiz0 = 0xb10
	pcgcctl   = 0xe00
	fifo0     = 0x1000
)

func r(off uintptr) uint32                 { return mmio.Load32(base + off) }
func w(off uintptr, v uint32)              { mmio.Store32(base+off, v) }
func bits(addr uintptr, clear, set uint32) { mmio.Store32(addr, mmio.Load32(addr)&^clear|set) }

var timer esp32s3.SystemTimer
var setup [8]byte
var started bool
var oldPHY uint32
var identity Config
var dirty bool
var sentAt uint32
var halted bool
var config byte
var report byte
var idle byte

// Every control response fits in one EP0 packet, including the longest string.
var device = []byte{18, 1, 0, 2, 0, 0, 0, 64, 0x3a, 0x30, 0, 0x80, 0, 1, 1, 2, 3, 1}

// Generic Desktop/Game Pad, two one-bit Button usages, six constant bits.
var hidReport = []byte{5, 1, 9, 5, 0xa1, 1, 5, 9, 0x19, 1, 0x29, 2, 0x15, 0, 0x25, 1, 0x75, 1, 0x95, 2, 0x81, 2, 0x75, 6, 0x95, 1, 0x81, 3, 0xc0}

// Keyboard: modifier bits, one reserved byte, then six key usage codes.
// This is a report-protocol keyboard, with no output LED report or boot mode.
var keyboardDescriptor = []byte{
	0x05, 0x01, 0x09, 0x06, 0xa1, 0x01,
	0x05, 0x07, 0x19, 0xe0, 0x29, 0xe7, 0x15, 0x00, 0x25, 0x01,
	0x75, 0x01, 0x95, 0x08, 0x81, 0x02,
	0x75, 0x08, 0x95, 0x01, 0x81, 0x03,
	0x19, 0x00, 0x29, 0x65, 0x15, 0x00, 0x25, 0x65,
	0x75, 0x08, 0x95, 0x06, 0x81, 0x00, 0xc0,
}
var keyboardPacket [8]byte
var configuration = []byte{9, 2, 34, 0, 1, 1, 0, 0x80, 50, 9, 4, 0, 0, 1, 3, 0, 0, 0, 9, 0x21, 0x11, 1, 0, 1, 0x22, 29, 0, 7, 5, 0x81, 3, 1, 0, 10}
var scratch [64]byte

func wait(off uintptr, mask, want uint32) bool {
	start := timer.Ticks()
	for r(off)&mask != want {
		if timer.Ticks()-start > 1600000 {
			return false
		}
	}
	return true
}
func flush() {
	w(grstctl, 1<<5|0x10<<6)
	wait(grstctl, 1<<5, 0)
	w(grstctl, 1<<4)
	wait(grstctl, 1<<4, 0)
}
func armOut() {
	w(doeptsiz0, 3<<29|1<<19|64)
	w(doepctl0, r(doepctl0)&^uint32(1<<21)|1<<31|1<<26)
}
func reset() {
	config = 0
	dirty = true
	halted = false
	for i := uintptr(0); i < 7; i++ {
		w(doepctl0+i*32, 1<<27)
		w(diepctl0+i*32, 1<<27)
		w(diepint0+i*32, 0xffffffff)
		w(doepint0+i*32, 0xffffffff)
	}
	flush()
	// Partition the 256-word FIFO: RX [0,128), EP0 [128,192), EP1 [192,208).
	w(grxfsiz, 128)
	w(dieptxf0, 64<<16|128)
	w(dieptxf1, 16<<16|192)
	w(gdfifocfg, 256<<16|256)
	w(dcfg, 3|1<<2)
	w(diepmsk, 1)
	w(doepmsk, 9)
	w(daintmsk, 1<<16|3)
	w(diepctl0, 1<<15)
	w(doepctl0, 1<<15)
	armOut()
}
func send(ep uintptr, data []byte) {
	off := uintptr(diepctl0) + ep*32
	w(off+8, 0xffffffff)
	w(off+16, 1<<19|uint32(len(data)))
	w(off, r(off)&^uint32(1<<21)|1<<26|1<<31)
	for i := 0; i < len(data); i += 4 {
		value := uint32(0)
		for j := 0; j < 4 && i+j < len(data); j++ {
			value |= uint32(data[i+j]) << uint(j*8)
		}
		w(fifo0+ep*0x1000, value)
	}
}
func reply(data []byte) {
	length := int(setup[6]) | int(setup[7])<<8
	if len(data) > length {
		data = data[:length]
	}
	send(0, data)
}
func stringReply(s string) {
	n := len(s)
	if n > 31 {
		n = 31
	}
	scratch[0] = byte(2 + n*2)
	scratch[1] = 3
	for i := 0; i < n; i++ {
		scratch[2+i*2] = s[i]
		scratch[3+i*2] = 0
	}
	reply(scratch[:2+n*2])
}
func handleSetup() {
	typ, req := setup[0], setup[1]
	value := uint16(setup[2]) | uint16(setup[3])<<8
	index := uint16(setup[4]) | uint16(setup[5])<<8
	w(doepint0, 0xffffffff)
	if typ == 0x80 && req == 6 {
		switch value >> 8 {
		case 1:
			reply(device)
			return
		case 2:
			reply(configuration)
			return
		case 3:
			switch byte(value) {
			case 0:
				scratch[0] = 4
				scratch[1] = 3
				scratch[2] = 9
				scratch[3] = 4
				reply(scratch[:4])
				return
			case 1:
				stringReply(identity.Manufacturer)
				return
			case 2:
				stringReply(identity.Product)
				return
			case 3:
				stringReply(identity.Serial)
				return
			}
		}
	}
	if typ == 0x81 && req == 6 && index == 0 {
		if value == 0x2200 {
			if identity.Keyboard {
				reply(keyboardDescriptor)
			} else {
				reply(hidReport)
			}
			return
		}
		if value == 0x2100 {
			reply(configuration[18:27])
			return
		}
	}
	if typ == 0 && req == 5 && value < 128 {
		w(dcfg, (r(dcfg)&^uint32(127<<4))|uint32(value)<<4)
		reply(scratch[:0])
		return
	}
	if typ == 0 && req == 9 && value <= 1 {
		config = byte(value)
		dirty = true
		halted = false
		if config == 1 {
			w(diepctl1, uint32(configuration[31])|1<<15|3<<18|1<<22|1<<28|1<<27)
		} else {
			w(diepctl1, r(diepctl1)|1<<30|1<<27)
		}
		reply(scratch[:0])
		return
	}
	if typ == 0x80 && req == 8 {
		scratch[0] = config
		reply(scratch[:1])
		return
	}
	if req == 0 && ((typ == 0x80 && index == 0) || (typ == 0x81 && index == 0) || (typ == 0x82 && (index == 0 || index == 0x80 || index == 0x81))) {
		scratch[0] = 0
		if typ == 0x82 && index == 0x81 && halted {
			scratch[0] = 1
		}
		scratch[1] = 0
		reply(scratch[:2])
		return
	}
	if typ == 2 && (req == 1 || req == 3) && value == 0 && index == 0x81 && config == 1 {
		halted = req == 3
		if halted {
			w(diepctl1, r(diepctl1)|1<<21)
		} else {
			w(diepctl1, r(diepctl1)&^uint32(1<<21)|1<<28)
			dirty = true
		}
		reply(scratch[:0])
		return
	}
	if typ == 0x81 && req == 10 && index == 0 {
		scratch[0] = 0
		reply(scratch[:1])
		return
	}
	if typ == 1 && req == 11 && index == 0 && value == 0 {
		reply(scratch[:0])
		return
	}
	if typ == 0x21 && req == 10 && index == 0 {
		idle = byte(value >> 8)
		reply(scratch[:0])
		return
	}
	if typ == 0xa1 && req == 2 && index == 0 {
		scratch[0] = idle
		reply(scratch[:1])
		return
	}
	if typ == 0xa1 && req == 1 && index == 0 && value == 0x100 {
		reply(inputReport())
		return
	}
	w(diepctl0, r(diepctl0)|1<<21)
	w(doepctl0, r(doepctl0)|1<<21)
}
func pollController() {
	status := r(gintsts)
	if status&(1<<12) != 0 {
		w(gintsts, 1<<12)
		reset()
		return
	}
	if status&(1<<13) != 0 {
		w(gintsts, 1<<13)
	}
	for i := 0; i < 16 && r(gintsts)&(1<<4) != 0; i++ {
		rx := r(grxstsp)
		count := int(rx >> 4 & 0x7ff)
		kind := rx >> 17 & 15
		for n := 0; n < count; n += 4 {
			word := r(fifo0)
			for j := 0; j < 4 && n+j < count; j++ {
				if kind == 6 && n+j < 8 {
					setup[n+j] = byte(word >> uint(j*8))
				}
			}
		}
		if kind == 4 && rx&15 == 0 {
			handleSetup()
		}
	}
	if r(diepint0)&1 != 0 {
		w(diepint0, 1)
		armOut()
	}
	if r(doepint0)&1 != 0 {
		w(doepint0, 1)
		armOut()
	}
	if r(diepint1)&1 != 0 {
		w(diepint1, 1)
	}
}
func initUSB() bool {
	// Enable/reset OTG, route the internal PHY away from Serial/JTAG, and
	// supply device-mode ID/VBUS signals through the GPIO input matrix.
	bits(0x600c0018, 0, 1<<23)
	bits(0x600c0020, 0, 1<<23)
	bits(0x600c0020, 1<<23, 0)
	bits(0x60039000, 1<<2|1<<5|1<<6|1<<11|1<<12, 1<<18|1<<19|1<<20|1<<31)
	bits(0x60008120, 0, 1<<20|1<<19)
	for i := uintptr(58); i <= 61; i++ {
		v := uint32(0x38 | 1<<7)
		if i == 59 {
			v = 0x3c | 1<<7
		}
		mmio.Store32(0x60004154+i*4, v)
	}
	// Select the 48 MHz full-speed PHY before resetting the DWC2 core.
	w(pcgcctl, 0)
	w(gahbcfg, 0)
	w(gusbcfg, 1<<6)
	if !wait(grstctl, 1<<31, 1<<31) {
		return false
	}
	w(grstctl, 1)
	if !wait(grstctl, 1, 0) {
		return false
	}
	timer.DelayMicroseconds(10)
	w(gusbcfg, 1<<6|5<<10|7|1<<30)
	w(dctl, 2)
	timer.DelayMilliseconds(25)
	w(gotgctl, 0xc0)
	w(dcfg, 7)
	flush()
	w(gintsts, 0xffffffff)
	w(gotgint, 0xffffffff)
	w(gintmsk, 1<<4|1<<12|1<<13|1<<18|1<<19)
	w(gahbcfg, 1)
	w(dctl, 0)
	return true
}

// Config supplies the device identity. Strings must be ASCII and at most 31
// bytes. VendorID and ProductID must be appropriate for the attached hardware.
type Config struct {
	VendorID, ProductID           uint16
	Manufacturer, Product, Serial string
	// Keyboard maps Poll bits 0 and 1 to the keyboard A and B usages.
	// The default exposes the bits as gamepad buttons instead.
	Keyboard bool
}

// Start connects a full-speed HID gamepad or keyboard. Only one
// controller is supported. Call Poll at least once per millisecond while active.
// The device is bus-powered, with a 100 mA maximum and no remote wakeup.
func Start(c Config) bool {
	if started || c.VendorID == 0 || !validString(c.Manufacturer) || !validString(c.Product) || !validString(c.Serial) {
		return false
	}
	identity = c
	configureReports(c.Keyboard)
	device[8] = byte(c.VendorID)
	device[9] = byte(c.VendorID >> 8)
	device[10] = byte(c.ProductID)
	device[11] = byte(c.ProductID >> 8)
	config = 0
	report = 0
	idle = 0
	dirty = true
	halted = false
	oldPHY = mmio.Load32(0x60008120)
	started = true
	if !initUSB() {
		Stop()
		return false
	}
	return true
}

func configureReports(keyboard bool) {
	configuration[25] = byte(len(hidReport))
	configuration[31] = 1
	if keyboard {
		configuration[25] = byte(len(keyboardDescriptor))
		configuration[31] = 8
	}
}

func inputReport() []byte {
	if !identity.Keyboard {
		scratch[0] = report
		return scratch[:1]
	}
	for i := 0; i < len(keyboardPacket); i++ {
		keyboardPacket[i] = 0
	}
	at := 2
	if report&1 != 0 {
		keyboardPacket[at] = 0x04 // Keyboard a/A.
		at++
	}
	if report&2 != 0 {
		keyboardPacket[at] = 0x05 // Keyboard b/B.
	}
	return keyboardPacket[:]
}
func validString(s string) bool {
	if len(s) > 31 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 32 || s[i] > 126 {
			return false
		}
	}
	return true
}

// Configured reports whether the host has selected the HID configuration.
func Configured() bool { return started && config == 1 }

// Poll services control requests and queues a report when button states change.
// Bits 0 and 1 represent buttons 1 and 2 (A and B in Keyboard mode).
// All other bits are ignored. Zero releases all keys.
// There is no allocation on this path. The caller must debounce physical inputs.
func Poll(buttons byte) {
	if !started {
		return
	}
	buttons &= 3
	if buttons != report {
		report = buttons
		dirty = true
	}
	pollController()
	now := timer.Ticks()
	if config != 1 || halted || r(dsts)&1 != 0 {
		return
	}
	if idle != 0 && now-sentAt >= uint32(idle)*64000 {
		dirty = true
	}
	if dirty && r(diepctl1)&(1<<31) == 0 {
		send(1, inputReport())
		dirty = false
		sentAt = now
	}
}

// Stop disconnects HID and gives the internal PHY back to its previous owner.
// A reboot may be needed to resume console output after changing USB function.
func Stop() {
	if !started {
		return
	}
	w(dctl, 2)
	w(gintmsk, 0)
	w(gahbcfg, 0)
	mmio.Store32(0x60008120, oldPHY)
	started = false
	config = 0
}
