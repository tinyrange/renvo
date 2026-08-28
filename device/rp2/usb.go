package rp2

import "renvo.dev/device/mmio"

const (
	usbDPRAM = uintptr(0x50100000)
	usbRegs  = uintptr(0x50110000)

	usbAddress           = usbRegs + 0x00
	usbMainControl       = usbRegs + 0x40
	usbSIEControl        = usbRegs + 0x4c
	usbSIEStatus         = usbRegs + 0x50
	usbBufferStatus      = usbRegs + 0x58
	usbEndpointAbort     = usbRegs + 0x60
	usbEndpointAbortDone = usbRegs + 0x64
	usbMuxing            = usbRegs + 0x74
	usbPower             = usbRegs + 0x78
	usbInterruptRaw      = usbRegs + 0x8c
	usbClearAlias        = uintptr(0x3000)

	bufferFull   = uint32(1 << 15)
	bufferLast   = uint32(1 << 14)
	bufferData1  = uint32(1 << 13)
	bufferSelect = uint32(1 << 12)
	bufferAvail  = uint32(1 << 10)

	endpointEnable    = uint32(1 << 31)
	endpointInterrupt = uint32(1 << 29)
	endpointBulk      = uint32(2 << 26)

	sieSetupReceived    = uint32(1 << 17)
	sieBusResetReceived = uint32(1 << 19)
	interruptSetup      = uint32(1 << 16)
	interruptBusReset   = uint32(1 << 12)

	ep1InOffset  = uint32(0x180)
	ep1OutOffset = uint32(0x1c0)
)

// USBDevice is the polling USB transport used by the resident reload monitor.
// Its controller and DPRAM layout is common to RP2040 and RP2350.
type USBDevice struct {
	configured  bool
	ep0InData1  bool
	ep0InBusy   bool
	ep0OutBusy  bool
	ep1InData1  bool
	ep1OutData1 bool
	inBusy      bool
	outReady    bool
	outLength   uint8
}

func usbBufferControl(endpoint uint8, in bool) uintptr {
	offset := uintptr(0x80) + uintptr(endpoint)*8
	if !in {
		offset += 4
	}
	return usbDPRAM + offset
}

func dpramCopyTo(offset uintptr, source []byte) {
	for index := 0; index < len(source); index += 4 {
		word := uint32(source[index])
		if index+1 < len(source) {
			word |= uint32(source[index+1]) << 8
		}
		if index+2 < len(source) {
			word |= uint32(source[index+2]) << 16
		}
		if index+3 < len(source) {
			word |= uint32(source[index+3]) << 24
		}
		mmio.Store32(usbDPRAM+offset+uintptr(index), word)
	}
}

func dpramCopyFrom(destination []byte, offset uintptr, count int) {
	for index := 0; index < count; index += 4 {
		word := mmio.Load32(usbDPRAM + offset + uintptr(index))
		destination[index] = byte(word)
		if index+1 < count {
			destination[index+1] = byte(word >> 8)
		}
		if index+2 < count {
			destination[index+2] = byte(word >> 16)
		}
		if index+3 < count {
			destination[index+3] = byte(word >> 24)
		}
	}
}

func armBuffer(control uintptr, value uint32) {
	mmio.Store32(control, value&^bufferAvail)
	// The controller needs at least 12 peripheral cycles between publishing
	// the control word and setting AVAILABLE. One Renvo MMIO load plus the
	// generated call sequence comfortably exceeds that bound; a source-level
	// loop makes control responses needlessly late on the boot clock.
	_ = mmio.Load32(usbMainControl)
	mmio.Store32(control, value)
}

func (d *USBDevice) sendEP0(data []byte) {
	if len(data) == 0 {
		d.sendEP0Status()
		return
	}
	if len(data) > 64 {
		data = data[:64]
	}
	dpramCopyTo(0x100, data)
	control := uint32(len(data)) | bufferFull | bufferLast | bufferSelect | bufferAvail
	if d.ep0InData1 {
		control |= bufferData1
	}
	d.ep0InData1 = !d.ep0InData1
	d.ep0InBusy = true
	armBuffer(usbBufferControl(0, true), control)
}

func (d *USBDevice) sendEP0Status() {
	control := bufferFull | bufferLast | bufferSelect | bufferAvail
	if d.ep0InData1 {
		control |= bufferData1
	}
	d.ep0InData1 = !d.ep0InData1
	d.ep0InBusy = true
	armBuffer(usbBufferControl(0, true), control)
}

func (d *USBDevice) armEP0Out() {
	d.ep0OutBusy = true
	armBuffer(usbBufferControl(0, false), 64|bufferLast|bufferData1|bufferSelect|bufferAvail)
}

func abortEndpoint(mask uint32, control uintptr) {
	mmio.Store32(usbEndpointAbort+0x2000, mask)
	// RP2040 B0/B1 do not implement EP_ABORT. Released RP2040 parts and
	// RP2350 acknowledge promptly, but a bound keeps the common image from
	// deadlocking if it is ever run on early silicon.
	for attempts := 0; attempts < 1024; attempts++ {
		if mmio.Load32(usbEndpointAbortDone)&mask != 0 {
			break
		}
	}
	mmio.Store32(control, 0)
	mmio.Store32(usbEndpointAbortDone+usbClearAlias, mask)
	mmio.Store32(usbEndpointAbort+usbClearAlias, mask)
}

func (d *USBDevice) resetEP0() {
	if d.ep0InBusy {
		abortEndpoint(1, usbBufferControl(0, true))
		d.ep0InBusy = false
	}
	if d.ep0OutBusy {
		abortEndpoint(2, usbBufferControl(0, false))
		d.ep0OutBusy = false
	}
	d.ep0InData1 = true
}

func stringDescriptor(text string, result *[64]byte) []byte {
	count := len(text)
	if count > 31 {
		count = 31
	}
	result[0] = byte(2 + count*2)
	result[1] = 3
	for index := 0; index < count; index++ {
		result[2+index*2] = text[index]
		result[3+index*2] = 0
	}
	return result[:2+count*2]
}

func (d *USBDevice) setup() {
	setup0 := mmio.Load32(usbDPRAM)
	setup1 := mmio.Load32(usbDPRAM + 4)
	// Acknowledge this request after capturing both words and before arming its
	// response, so a subsequent request cannot be erased by this clear.
	mmio.Store32(usbSIEStatus+usbClearAlias, sieSetupReceived)
	requestType := byte(setup0)
	request := byte(setup0 >> 8)
	value := uint16(setup0 >> 16)
	length := int(setup1 >> 16)
	d.resetEP0()
	mmio.Store32(usbBufferControl(0, true), bufferData1|bufferSelect)
	mmio.Store32(usbBufferControl(0, false), bufferData1|bufferSelect)

	if requestType == 0x80 && request == 6 {
		kind := byte(value >> 8)
		index := byte(value)
		if kind == 1 {
			mmio.Store32(usbDPRAM+0x100, 0x02000112)
			mmio.Store32(usbDPRAM+0x104, 0x40000000)
			mmio.Store32(usbDPRAM+0x108, 0x4021cafe)
			mmio.Store32(usbDPRAM+0x10c, 0x00000100)
			mmio.Store32(usbDPRAM+0x110, 0x00000100)
			if length > 18 {
				length = 18
			}
			control := uint32(length) | bufferFull | bufferLast | bufferSelect | bufferAvail
			if d.ep0InData1 {
				control |= bufferData1
			}
			d.ep0InData1 = !d.ep0InData1
			d.ep0InBusy = true
			armBuffer(usbBufferControl(0, true), control)
			return
		} else if kind == 2 {
			mmio.Store32(usbDPRAM+0x100, 0x00200209)
			mmio.Store32(usbDPRAM+0x104, 0x80000101)
			mmio.Store32(usbDPRAM+0x108, 0x00040932)
			mmio.Store32(usbDPRAM+0x10c, 0x52ff0200)
			mmio.Store32(usbDPRAM+0x110, 0x05070001)
			mmio.Store32(usbDPRAM+0x114, 0x00400201)
			mmio.Store32(usbDPRAM+0x118, 0x81050700)
			mmio.Store32(usbDPRAM+0x11c, 0x00004002)
			if length > 32 {
				length = 32
			}
			control := uint32(length) | bufferFull | bufferLast | bufferSelect | bufferAvail
			if d.ep0InData1 {
				control |= bufferData1
			}
			d.ep0InData1 = !d.ep0InData1
			d.ep0InBusy = true
			armBuffer(usbBufferControl(0, true), control)
			return
		} else if kind == 3 && index == 0 {
			var scratch [64]byte
			scratch[0], scratch[1], scratch[2], scratch[3] = 4, 3, 0x09, 0x04
			d.sendEP0(scratch[:4])
			return
		}
		d.sendEP0Status()
		return
	}
	if requestType == 0 && request == 5 {
		control := bufferFull | bufferLast | bufferSelect | bufferAvail
		if d.ep0InData1 {
			control |= bufferData1
		}
		d.ep0InData1 = !d.ep0InData1
		d.ep0InBusy = true
		armBuffer(usbBufferControl(0, true), control)
		// The host may issue its first token at the new address immediately after
		// acknowledging this status packet. Complete the address handoff here;
		// returning to the general polling loop is too slow at the boot clock.
		for mmio.Load32(usbBufferStatus)&1 == 0 {
		}
		mmio.Store32(usbBufferStatus+usbClearAlias, 1)
		d.ep0InBusy = false
		mmio.Store32(usbAddress, uint32(byte(value)))
		return
	}
	if requestType == 0 && request == 9 {
		d.configured = value != 0
		if d.configured {
			d.openVendorEndpoints()
		}
		control := bufferFull | bufferLast | bufferSelect | bufferAvail
		if d.ep0InData1 {
			control |= bufferData1
		}
		d.ep0InData1 = !d.ep0InData1
		d.ep0InBusy = true
		armBuffer(usbBufferControl(0, true), control)
		return
	}
	if requestType == 0x80 && request == 8 {
		value := byte(0)
		if d.configured {
			value = 1
		}
		d.sendEP0([]byte{value})
		return
	}
	if requestType == 0x80 && request == 0 {
		d.sendEP0([]byte{0, 0})
		return
	}
	// Unsupported requests are acknowledged with an empty response while the
	// monitor descriptor surface stays deliberately minimal.
	control := bufferFull | bufferLast | bufferSelect | bufferAvail
	if d.ep0InData1 {
		control |= bufferData1
	}
	d.ep0InData1 = !d.ep0InData1
	d.ep0InBusy = true
	armBuffer(usbBufferControl(0, true), control)
}

func (d *USBDevice) openVendorEndpoints() {
	mmio.Store32(usbDPRAM+0x08, endpointEnable|endpointInterrupt|endpointBulk|ep1InOffset)
	mmio.Store32(usbDPRAM+0x0c, endpointEnable|endpointInterrupt|endpointBulk|ep1OutOffset)
	d.ep1InData1, d.ep1OutData1, d.inBusy = false, false, false
	d.armVendorOut()
}

func (d *USBDevice) armVendorOut() {
	// Do not hand the buffer back until the controller has observed the W1C
	// for its previous completion. Otherwise a fast polling loop can retire
	// the old status a second time and dispatch the DPRAM packet twice.
	for mmio.Load32(usbBufferStatus)&8 != 0 {
	}
	control := uint32(64) | bufferLast | bufferSelect | bufferAvail
	if d.ep1OutData1 {
		control |= bufferData1
	}
	d.ep1OutData1 = !d.ep1OutData1
	armBuffer(usbBufferControl(1, false), control)
}

// Start connects the already clocked USB controller to the on-chip PHY.
func (d *USBDevice) Start() {
	if isRP2350() {
		mmio.Store32(0x40022000, 1<<28)
		mmio.Store32(0x40023000, 1<<28)
		for mmio.Load32(0x40020008)&(1<<28) == 0 {
		}
	} else {
		mmio.Store32(0x4000e000, 1<<24)
		mmio.Store32(0x4000f000, 1<<24)
		for mmio.Load32(0x4000c008)&(1<<24) == 0 {
		}
	}
	for offset := uintptr(0); offset < 0x1000; offset += 4 {
		mmio.Store32(usbDPRAM+offset, 0)
	}
	mmio.Store32(usbMuxing, 0x09)
	mmio.Store32(usbPower, 0x0c)
	mmio.Store32(usbMainControl, 1)
	// EP0 uses one buffer in each direction; request completion after that
	// buffer rather than waiting for a nonexistent double-buffer pair.
	mmio.Store32(usbSIEControl, 1<<29)
	mmio.Store32(usbSIEControl, 1<<29|1<<16)
	d.configured = false
	d.ep0InBusy, d.ep0OutBusy = false, false
}

// Poll services enumeration and endpoint completion without native libraries.
func (d *USBDevice) Poll() {
	interrupts := mmio.Load32(usbInterruptRaw)
	if interrupts>>12&1 != 0 {
		mmio.Store32(usbSIEStatus+usbClearAlias, sieBusResetReceived)
		mmio.Store32(usbAddress, 0)
		d.configured, d.inBusy, d.outReady = false, false, false
	}
	// Retire transfers before accepting a new setup packet. Both interrupt
	// causes can be asserted together, and treating the new request first can
	// mistake the preceding control transfer's completion for the new status
	// stage (most visibly by applying SET_ADDRESS before its zero-length IN).
	buffers := mmio.Load32(usbBufferStatus)
	if buffers != 0 {
		if buffers&1 != 0 {
			completedLength := mmio.Load32(usbBufferControl(0, true)) & 0x3ff
			mmio.Store32(usbBufferStatus+usbClearAlias, 1)
			d.ep0InBusy = false
			// A nonempty IN packet is the data stage of a control read and
			// therefore needs an OUT status packet. A zero-length IN packet is
			// already the status stage of a control write.
			if completedLength != 0 {
				d.armEP0Out()
			}
		}
		if buffers&2 != 0 {
			mmio.Store32(usbBufferStatus+usbClearAlias, 2)
			d.ep0OutBusy = false
		}
		if buffers&4 != 0 {
			mmio.Store32(usbBufferStatus+usbClearAlias, 4)
			d.inBusy = false
		}
		if buffers&8 != 0 {
			mmio.Store32(usbBufferStatus+usbClearAlias, 8)
			d.outLength = uint8(mmio.Load32(usbBufferControl(1, false)) & 0x3ff)
			d.outReady = true
		}
	}
	// Re-read after retiring reset and buffer events. A setup packet can arrive
	// while those paths run, especially at the boot clock, and SETUP_REC must
	// be observed from the current controller state rather than the entry
	// snapshot.
	if mmio.Load32(usbInterruptRaw)>>16&1 != 0 {
		d.setup()
	}
}

// ReadPacket copies one received vendor packet. The caller must consume it
// before the endpoint is armed again.
func (d *USBDevice) ReadPacket(result []byte) int {
	if !d.outReady {
		return 0
	}
	count := int(d.outLength)
	if count > len(result) {
		count = len(result)
	}
	dpramCopyFrom(result, uintptr(ep1OutOffset), count)
	d.outReady = false
	d.armVendorOut()
	return count
}

// WritePacket queues one vendor response and reports whether the endpoint was free.
func (d *USBDevice) WritePacket(data []byte) bool {
	if d.inBusy || !d.configured {
		return false
	}
	if len(data) > 64 {
		data = data[:64]
	}
	dpramCopyTo(uintptr(ep1InOffset), data)
	control := uint32(len(data)) | bufferFull | bufferLast | bufferSelect | bufferAvail
	if d.ep1InData1 {
		control |= bufferData1
	}
	d.ep1InData1 = !d.ep1InData1
	d.inBusy = true
	armBuffer(usbBufferControl(1, true), control)
	return true
}
