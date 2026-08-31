//go:build m5papermonolite

package board

import (
	"renvo.dev/device/display/ssd1677"
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
	"renvo.dev/device/mmio"
)

const (
	spi2Base        = uintptr(0x60024000)
	spi2Command     = spi2Base + 0x00
	spi2Control     = spi2Base + 0x08
	spi2Clock       = spi2Base + 0x0c
	spi2User        = spi2Base + 0x10
	spi2DataLength  = spi2Base + 0x1c
	spi2Misc        = spi2Base + 0x20
	spi2Data        = spi2Base + 0x98
	spi2ClockGate   = spi2Base + 0xe8
	spiSystemClock0 = uintptr(0x600c0018)
	spiSystemReset0 = uintptr(0x600c0020)

	spi2Enable    = uint32(1 << 6)
	spi2Update    = uint32(1 << 23)
	spi2UserStart = uint32(1 << 24)
	spi2UserMOSI  = uint32(1 << 27)

	spi2MOSIOutputSignal  = uint32(103)
	spi2ClockOutputSignal = uint32(101)
	spiWaitTimeoutMS      = uint32(10)
)

var epdMOSI = esp32s3.GPIO(14)
var epdClock = esp32s3.GPIO(15)
var epdDataCommand = esp32s3.GPIO(17)
var epdBusy = esp32s3.GPIO(18)

type paperEPDTransport struct {
	ready   bool
	writing bool
}

func (transport *paperEPDTransport) initialize() error {
	epdChipSelect.Set(true)
	if err := epdChipSelect.Configure(gpio.Config{Direction: gpio.Output}); err != nil {
		return err
	}
	epdDataCommand.Set(false)
	if err := epdDataCommand.Configure(gpio.Config{Direction: gpio.Output}); err != nil {
		return err
	}
	if err := epdBusy.Configure(gpio.Config{Direction: gpio.Input, Pull: gpio.PullUp}); err != nil {
		return err
	}
	if err := epdMOSI.ConfigureOutputSignal(spi2MOSIOutputSignal); err != nil {
		return err
	}
	if err := epdClock.ConfigureOutputSignal(spi2ClockOutputSignal); err != nil {
		return err
	}

	mmio.Store32(spiSystemClock0, mmio.Load32(spiSystemClock0)|spi2Enable)
	mmio.Store32(spiSystemReset0, mmio.Load32(spiSystemReset0)|spi2Enable)
	mmio.Store32(spiSystemReset0, mmio.Load32(spiSystemReset0)&^spi2Enable)
	mmio.Store32(spi2ClockGate, 7)
	mmio.Store32(spi2Control, 0)
	// The 80 MHz peripheral clock divided by four gives the official 20 MHz
	// mode-0 write clock (N=3, low=3, high=1).
	mmio.Store32(spi2Clock, uint32(3|(1<<6)|(3<<12)))
	mmio.Store32(spi2User, spi2UserMOSI)
	// Keep hardware chip-select outputs disconnected. CS and D/C remain GPIOs.
	mmio.Store32(spi2Misc, 0x3f)
	mmio.Store32(spi2Command, spi2Update)
	if !transport.wait(spi2Update) {
		return ErrDisplaySPI
	}
	transport.ready = true
	return nil
}

func (transport *paperEPDTransport) Reset() error { return Power.ResetDisplay() }

func (transport *paperEPDTransport) Begin(command byte) error {
	if !transport.ready || transport.writing {
		return ErrDisplaySPI
	}
	epdChipSelect.Set(false)
	epdDataCommand.Set(false)
	transport.writing = true
	if err := transport.write([]byte{command}); err != nil {
		transport.abort()
		return err
	}
	epdDataCommand.Set(true)
	return nil
}

func (transport *paperEPDTransport) Data(data []byte) error {
	if !transport.writing {
		return ErrDisplaySPI
	}
	return transport.write(data)
}

func (transport *paperEPDTransport) End() error {
	if !transport.writing {
		return ErrDisplaySPI
	}
	epdChipSelect.Set(true)
	transport.writing = false
	return nil
}

func (transport *paperEPDTransport) Busy() bool { return epdBusy.Get() }
func (transport *paperEPDTransport) Milliseconds() uint32 {
	return Clock.Milliseconds()
}
func (transport *paperEPDTransport) DelayMilliseconds(milliseconds uint32) {
	Clock.DelayMilliseconds(milliseconds)
}

func (transport *paperEPDTransport) abort() {
	epdChipSelect.Set(true)
	transport.writing = false
}

func (transport *paperEPDTransport) deactivate() {
	transport.ready = false
	transport.abort()
}

func (transport *paperEPDTransport) wait(mask uint32) bool {
	started := Clock.Milliseconds()
	for mmio.Load32(spi2Command)&mask != 0 {
		if Clock.Milliseconds()-started >= spiWaitTimeoutMS {
			return false
		}
	}
	return true
}

func (transport *paperEPDTransport) write(data []byte) error {
	for len(data) != 0 {
		count := len(data)
		if count > 64 {
			count = 64
		}
		for offset := 0; offset < count; offset += 4 {
			word := uint32(0)
			wordLength := count - offset
			if wordLength > 4 {
				wordLength = 4
			}
			for byteOffset := 0; byteOffset < wordLength; byteOffset++ {
				word |= uint32(data[offset+byteOffset]) << uint(byteOffset*8)
			}
			spi2StoreWord(offset/4, word)
		}
		mmio.Store32(spi2DataLength, uint32(count*8-1))
		mmio.Store32(spi2Command, spi2Update)
		if !transport.wait(spi2Update) {
			transport.abort()
			return ErrDisplaySPI
		}
		mmio.Store32(spi2Command, spi2UserStart)
		if !transport.wait(spi2UserStart) {
			transport.abort()
			return ErrDisplaySPI
		}
		data = data[count:]
	}
	return nil
}

func spi2StoreWord(index int, value uint32) {
	switch index {
	case 0:
		mmio.Store32(spi2Data+0x00, value)
	case 1:
		mmio.Store32(spi2Data+0x04, value)
	case 2:
		mmio.Store32(spi2Data+0x08, value)
	case 3:
		mmio.Store32(spi2Data+0x0c, value)
	case 4:
		mmio.Store32(spi2Data+0x10, value)
	case 5:
		mmio.Store32(spi2Data+0x14, value)
	case 6:
		mmio.Store32(spi2Data+0x18, value)
	case 7:
		mmio.Store32(spi2Data+0x1c, value)
	case 8:
		mmio.Store32(spi2Data+0x20, value)
	case 9:
		mmio.Store32(spi2Data+0x24, value)
	case 10:
		mmio.Store32(spi2Data+0x28, value)
	case 11:
		mmio.Store32(spi2Data+0x2c, value)
	case 12:
		mmio.Store32(spi2Data+0x30, value)
	case 13:
		mmio.Store32(spi2Data+0x34, value)
	case 14:
		mmio.Store32(spi2Data+0x38, value)
	default:
		mmio.Store32(spi2Data+0x3c, value)
	}
}

type displayProtocol interface {
	FullMonochrome(frame []byte) error
	PartialMonochrome(frame []byte) error
	FullGray(plane1, plane2 []byte) error
	FullGrayStream(source ssd1677.GrayPlaneSource) error
	InvalidateBaseline()
}

type displayTransport interface {
	initialize() error
	deactivate()
}

type displayPower interface {
	EnableDisplayAndTouch() error
	DisableDisplayAndTouch() error
}

// PaperDisplay owns the PaperMono-Lite display power and transport policy.
type PaperDisplay struct {
	transport displayTransport
	protocol  displayProtocol
	power     displayPower
	active    bool
}

var paperDisplayTransport = paperEPDTransport{}

// Display is the PaperMono-Lite SSD1677 display.
var Display = newPaperDisplay(&paperDisplayTransport, ssd1677.New(&paperDisplayTransport), Power)

func newPaperDisplay(transport displayTransport, protocol displayProtocol, power displayPower) *PaperDisplay {
	return &PaperDisplay{transport: transport, protocol: protocol, power: power}
}

// Enable powers the panel and touch rail, then configures the bounded 20 MHz
// SPI2 mode-0 transport. It sends no SSD1677 command or refresh by itself.
func (display *PaperDisplay) Enable() error {
	if display.active {
		return nil
	}
	if err := display.power.EnableDisplayAndTouch(); err != nil {
		return err
	}
	if err := display.transport.initialize(); err != nil {
		_ = display.power.DisableDisplayAndTouch()
		return err
	}
	display.active = true
	return nil
}

// FullMonochrome performs an OTP full refresh from one 48,000-byte packed plane.
func (display *PaperDisplay) FullMonochrome(frame []byte) error {
	if err := display.Enable(); err != nil {
		return err
	}
	if err := display.protocol.FullMonochrome(frame); err != nil {
		return display.fail(err)
	}
	return nil
}

// PartialMonochrome performs a bounded OTP partial refresh. A prior successful
// full monochrome refresh is required; after ten differential updates the
// protocol automatically makes the next request a recovery full refresh.
func (display *PaperDisplay) PartialMonochrome(frame []byte) error {
	if err := display.Enable(); err != nil {
		return err
	}
	if err := display.protocol.PartialMonochrome(frame); err != nil {
		return display.fail(err)
	}
	return nil
}

// FullGray performs a full four-level OTP refresh from two packed bit planes.
func (display *PaperDisplay) FullGray(plane1, plane2 []byte) error {
	if err := display.Enable(); err != nil {
		return err
	}
	if err := display.protocol.FullGray(plane1, plane2); err != nil {
		return display.fail(err)
	}
	return nil
}

// FullGrayStream performs a full four-level OTP refresh from a packed-row
// source, avoiding two simultaneously resident 48,000-byte planes.
func (display *PaperDisplay) FullGrayStream(source ssd1677.GrayPlaneSource) error {
	if err := display.Enable(); err != nil {
		return err
	}
	if err := display.protocol.FullGrayStream(source); err != nil {
		return display.fail(err)
	}
	return nil
}

// Shutdown asserts reset and removes display/touch power. It invalidates the
// partial-refresh baseline even if the board-level shutdown reports an error.
func (display *PaperDisplay) Shutdown() error {
	display.protocol.InvalidateBaseline()
	display.active = false
	display.transport.deactivate()
	return display.power.DisableDisplayAndTouch()
}

func (display *PaperDisplay) fail(err error) error {
	_ = display.Shutdown()
	return err
}

type displayError string

func (err displayError) Error() string { return string(err) }

// ErrDisplaySPI reports a bounded SPI2 setup or transfer failure.
const ErrDisplaySPI displayError = "PaperMono-Lite SPI2 transfer failed"
