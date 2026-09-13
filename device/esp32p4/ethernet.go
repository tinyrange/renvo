package esp32p4

import (
	"renvo.dev/device/gpio"
	"renvo.dev/device/mmio"
)

const (
	emacBase      = uintptr(0x50098000)
	emacDMA       = emacBase + 0x1000
	ethBufferSize = 1536
	ethRXCount    = 8
	// EthernetDMASize is the required size of a dedicated, 64-byte-aligned
	// internal SRAM reservation. Never access it through the cached alias.
	EthernetDMASize = (ethRXCount + 1) * (64 + ethBufferSize)
)

type ethernetError string

func (e ethernetError) Error() string { return string(e) }

// EthernetConfig describes an IP101 PHY using the P4's RMII IO mux.
// RXDV/RXD0/RXD1 are GPIO28/29/30; TXD0/TXD1/TXEN are GPIO34/35/49;
// the external 50 MHz reference clock is GPIO50.
type EthernetConfig struct {
	MAC                   [6]byte
	MDC, MDIO, Reset, PHY uint8
	// DMA is a dedicated physical SRAM address, aligned to 64 bytes, with
	// EthernetDMASize bytes reserved by the caller for this driver's lifetime.
	DMA uintptr
}

// Ethernet owns the single ESP32-P4 MAC. Call its methods from one polling loop.
// DMA memory is supplied by the board and accessed through its uncached alias.
type Ethernet struct {
	config          EthernetConfig
	timer           SystemTimer
	rx, tx, buffers uintptr
	next            int
	// Received and Dropped count completed RX descriptors consumed by Receive.
	Received, Dropped uint32
	// LastRXStatus is the most recently consumed hardware descriptor status.
	LastRXStatus uint32
}

func ethReplace(address uintptr, clear, set uint32) {
	mmio.Store32(address, mmio.Load32(address)&^clear|set)
}

func (e *Ethernet) wait(address uintptr, mask uint32, set bool, ms uint32) bool {
	start := e.timer.Ticks()
	for (mmio.Load32(address)&mask != 0) != set {
		if e.timer.Ticks()-start >= ms*16000 {
			return false
		}
	}
	return true
}

// ReadPHY reads a Clause 22 management register with a bounded timeout.
func (e *Ethernet) ReadPHY(reg uint8) (uint16, error) {
	mmio.Store32(emacBase+0x10, uint32(e.config.PHY)<<11|uint32(reg&31)<<6|5<<2|1)
	if !e.wait(emacBase+0x10, 1, false, 10) {
		return 0, ethernetError("MDIO read timeout")
	}
	return uint16(mmio.Load32(emacBase + 0x14)), nil
}

func (e *Ethernet) writePHY(reg uint8, value uint16) error {
	mmio.Store32(emacBase+0x14, uint32(value))
	mmio.Store32(emacBase+0x10, uint32(e.config.PHY)<<11|uint32(reg&31)<<6|5<<2|3)
	if !e.wait(emacBase+0x10, 1, false, 10) {
		return ethernetError("MDIO write timeout")
	}
	return nil
}

// Configure resets the PHY and MAC and waits up to five seconds for 100/full
// negotiation. This initial driver advertises only that mode.
func (e *Ethernet) Configure(config EthernetConfig) error {
	if config.PHY > 31 || config.MDC > 54 || config.MDIO > 54 || config.Reset > 54 || config.MAC[0]&1 != 0 {
		return ethernetError("invalid Ethernet configuration")
	}
	physical := config.DMA
	if physical&63 != 0 || physical < 0x4ff00000 || physical > 0x4ff80000-EthernetDMASize {
		return ethernetError("Ethernet DMA requires internal SRAM")
	}
	cacheSize := mmio.Load32(0x3ff10278)
	if cacheSize != 1<<9 && cacheSize != 1<<10 {
		return ethernetError("unsupported L2 cache/SRAM partition")
	}
	e.rx = 0
	e.tx = 0
	e.Received = 0
	e.Dropped = 0
	e.LastRXStatus = 0
	e.config = config
	reset := GPIO(config.Reset)
	reset.Configure(gpio.Config{Direction: gpio.Output})
	reset.Set(false)
	e.timer.DelayMilliseconds(10)
	reset.Set(true)
	e.timer.DelayMilliseconds(100)
	// ESP-IDF esp32p4 emac_ll.h: bus gate, module reset, RMII clock input.
	ethReplace(0x500e6018, 0, 1<<13)
	ethReplace(0x5011104c, 0, 1<<30)
	ethReplace(0x5011104c, 1<<30, 0)
	ethReplace(0x500e514c, 7<<2, 4<<2)
	ethReplace(0x500e6030, 0x3f<<24, 1<<27|1<<29)
	ethReplace(0x500e6034, 0x3ffff, 1|1<<9|1<<10)
	ethReplace(0x50111040, 3<<13, 1<<15)
	ethReplace(0x500e60b4, 0, 1<<17)
	for _, n := range []uint8{28, 29, 30, 34, 35, 49, 50} {
		p := GPIO(n)
		input := n == 28 || n == 29 || n == 30 || n == 50
		v := mmio.Load32(p.ioMux()) &^ (gpioFunctionMask | gpioPullUp | gpioPullDown | gpioInputEnable)
		v |= 3 << 12
		if input {
			v |= gpioInputEnable
		}
		mmio.Store32(p.ioMux(), v)
		p.enable(!input)
	}
	for _, signal := range []uintptr{178, 179, 180} {
		ethReplace(gpioBase+0x158+signal*4, 1<<7, 0)
	}
	mdc := GPIO(config.MDC)
	mdc.Configure(gpio.Config{Direction: gpio.Output})
	mmio.Store32(mdc.outputSelect(), 108)
	mdio := GPIO(config.MDIO)
	mdio.Configure(gpio.Config{Direction: gpio.Input})
	mmio.Store32(mdio.outputSelect(), 109)
	mdio.enable(true)
	mmio.Store32(gpioBase+0x304, uint32(config.MDIO)|1<<7)
	mmio.Store32(emacDMA, 1)
	if !e.wait(emacDMA, 1, false, 100) {
		return ethernetError("EMAC reset timeout: check RMII clock")
	}
	id1, err := e.ReadPHY(2)
	if err != nil {
		return err
	}
	id2, err := e.ReadPHY(3)
	if err != nil {
		return err
	}
	if id1 != 0x243 || id2&0xfff0 != 0x0c50 {
		return ethernetError("IP101 PHY not found")
	}
	if err := e.writePHY(4, 0x101); err != nil {
		return err
	}
	if err := e.writePHY(0, 0x1200); err != nil {
		return err
	}
	start := e.timer.Ticks()
	for {
		status, err := e.ReadPHY(1)
		if err != nil {
			return err
		}
		if status&0x24 == 0x24 {
			break
		}
		if e.timer.Ticks()-start > 80000000 {
			return ethernetError("Ethernet link negotiation timeout")
		}
		e.timer.DelayMilliseconds(10)
	}
	// Chained enhanced descriptors; 64-byte aligned to avoid cache sharing.
	e.rx = physical + 0x40000000
	for offset := uintptr(0); offset < EthernetDMASize; offset += 4 {
		mmio.Store32(e.rx+offset, 0)
	}
	e.tx = e.rx + ethRXCount*64
	e.buffers = e.tx + 64
	for i := 0; i < ethRXCount; i++ {
		d := e.rx + uintptr(i)*64
		mmio.Store32(d+4, 1<<14|ethBufferSize)
		mmio.Store32(d+8, uint32(e.buffers+uintptr(i)*ethBufferSize-0x40000000))
		mmio.Store32(d+12, uint32(e.rx+uintptr((i+1)%ethRXCount)*64-0x40000000))
		mmio.Store32(d, 1<<31)
	}
	mmio.Store32(e.tx+8, uint32(e.buffers+ethRXCount*ethBufferSize-0x40000000))
	mmio.Store32(e.tx+12, uint32(e.tx-0x40000000))
	mmio.Store32(e.tx, 1<<20)
	mmio.Store32(emacDMA, 1<<25|1<<7|4<<8)
	mmio.Store32(emacDMA+12, uint32(e.rx-0x40000000))
	mmio.Store32(emacDMA+16, uint32(e.tx-0x40000000))
	mmio.Store32(emacBase+0x40, uint32(config.MAC[4])|uint32(config.MAC[5])<<8)
	mmio.Store32(emacBase+0x44, uint32(config.MAC[0])|uint32(config.MAC[1])<<8|uint32(config.MAC[2])<<16|uint32(config.MAC[3])<<24)
	mmio.Store32(emacBase+4, 0)
	mmio.Store32(emacBase+0x3c, 0xffffffff)
	mmio.Store32(emacDMA+0x1c, 0)
	mmio.Store32(emacBase, 1<<15|1<<14|1<<11|1<<3|1<<2)
	// The P4 RX FIFO is only 256 bytes: leave RSF (bit 25) CLEAR.
	// DT (bit 26) disables hardware checksum-error dropping; software validates
	// the IP and transport checksums. RX and TX run with 64-byte thresholds.
	// OSF (bit 2) must stay clear with a single self-chained TX descriptor:
	// prefetching the next frame before ownership writeback can send it twice.
	mmio.Store32(emacDMA+0x18, 1<<26|1<<13|1<<1)
	e.next = 0
	return nil
}

// Send transmits one Ethernet frame (without FCS), waiting for DMA completion.
func (e *Ethernet) Send(frame []byte) error {
	if e.tx == 0 || len(frame) < 14 || len(frame) > 1514 {
		return ethernetError("invalid Ethernet frame")
	}
	if !e.wait(e.tx, 1<<31, false, 100) {
		return ethernetError("Ethernet TX busy")
	}
	b := e.buffers + ethRXCount*ethBufferSize
	for i, v := range frame {
		mmio.Store8(b+uintptr(i), v)
	}
	mmio.Store32(e.tx+4, uint32(len(frame)))
	mmio.Store32(e.tx, 1<<31|1<<29|1<<28|1<<20)
	mmio.Store32(emacDMA+4, 0)
	if !e.wait(e.tx, 1<<31, false, 100) {
		return ethernetError("Ethernet TX timeout")
	}
	if mmio.Load32(e.tx)&(1<<15) != 0 {
		return ethernetError("Ethernet TX error")
	}
	return nil
}

// Receive copies the next frame without FCS, or returns zero when none is ready.
// Damaged, fragmented and oversized frames are discarded.
func (e *Ethernet) Receive(frame []byte) int {
	if e.rx == 0 {
		return 0
	}
	d := e.rx + uintptr(e.next)*64
	status := mmio.Load32(d)
	if status>>31 != 0 {
		return 0
	}
	e.LastRXStatus = status
	n := int(status>>16&0x3fff) - 4
	if status&(1<<15) == 0 && status&0x300 == 0x300 && n >= 14 && n <= len(frame) && n <= 1514 {
		b := e.buffers + uintptr(e.next)*ethBufferSize
		for i := 0; i < n; i++ {
			frame[i] = mmio.Load8(b + uintptr(i))
		}
	} else {
		n = 0
		e.Dropped++
	}
	if n > 0 {
		e.Received++
	}
	mmio.Store32(d, 1<<31)
	e.next = (e.next + 1) % ethRXCount
	mmio.Store32(emacDMA+8, 0)
	return n
}
