package esp32c6

import "renvo.dev/device/mmio"

const (
	usbSerialJTAGConf0 = uintptr(0x6000f018)
	usbSerialJTAGTest  = uintptr(0x6000f01c)

	usbPadPullOverride = uint32(1 << 8)
	usbDPPullUp        = uint32(1 << 9)
	usbDPPullDown      = uint32(1 << 10)
	usbDMPullUp        = uint32(1 << 11)
	usbDMPullDown      = uint32(1 << 12)
	usbPadEnable       = uint32(1 << 14)

	usbTestEnable = uint32(1)
	usbTestOutput = uint32(1 << 1)
	usbTestTXDP   = uint32(1 << 2)
	usbTestTXDM   = uint32(1 << 3)
	usbTestRXDP   = uint32(1 << 5)
	usbTestRXDM   = uint32(1 << 6)
)

// USBPHY exposes the ESP32-C6 USB Serial/JTAG block's documented raw full-
// speed PHY test interface. The software protocol may operate at low speed;
// no programmable USB controller is assumed.
type USBPHY struct {
	originalConf uint32
	bitDelay     uint32
	owned        bool
}

// NewUSBPHY returns a PHY using delayIterations in its cycle-counted bit-cell
// loop. Hardware bring-up must calibrate this value against the CPU clock.
func NewUSBPHY(delayIterations uint32) USBPHY {
	return USBPHY{bitDelay: delayIterations}
}

// Takeover selects the raw PHY test path, the internal low-speed D- pull-up,
// and preserves the configuration needed to restore USB Serial/JTAG.
func (p *USBPHY) Takeover() {
	if p.owned {
		return
	}
	p.originalConf = mmio.Load32(usbSerialJTAGConf0)
	config := p.originalConf &^ (usbDPPullUp | usbDPPullDown | usbDMPullUp | usbDMPullDown)
	config |= usbPadPullOverride | usbDMPullUp | usbPadEnable
	mmio.Store32(usbSerialJTAGConf0, config)
	mmio.Store32(usbSerialJTAGTest, usbTestEnable)
	p.owned = true
}

// Release restores the fixed USB Serial/JTAG function for programming and
// debugging. It is safe to call more than once.
func (p *USBPHY) Release() {
	if !p.owned {
		return
	}
	mmio.Store32(usbSerialJTAGTest, 0)
	mmio.Store32(usbSerialJTAGConf0, p.originalConf)
	p.owned = false
}

// Sample returns the single-ended D+ and D- receiver states.
func (p *USBPHY) Sample() (dp, dm bool) {
	value := mmio.Load32(usbSerialJTAGTest)
	return value&usbTestRXDP != 0, value&usbTestRXDM != 0
}

// Transmit drives pre-encoded low-speed line states at a fixed instruction
// cadence, then releases the PHY output while retaining the D- pull-up.
func (p *USBPHY) Transmit(states []byte) {
	if !p.owned {
		return
	}
	if len(states) > 128 {
		return
	}
	var values [128]uint32
	for index, line := range states {
		// LineJ is one and LineK is two, so the low bits map directly
		// onto the test interface's D-/D+ transmit bits.
		value := usbTestEnable | usbTestOutput
		value |= uint32(line&1) << 3
		value |= uint32((line>>1)&1) << 2
		values[index] = value
	}
	for index := 0; index < len(states); index++ {
		mmio.Store32(usbSerialJTAGTest, values[index])
		for delay := uint32(0); delay < p.bitDelay; delay++ {
		}
	}
	mmio.Store32(usbSerialJTAGTest, usbTestEnable)
}
