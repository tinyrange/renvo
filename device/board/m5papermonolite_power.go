//go:build m5papermonolite

package board

import (
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
	"renvo.dev/device/i2c"
	"renvo.dev/device/ioexpander/m5ioe1"
	"renvo.dev/device/power/m5pm1"
)

var internalI2CController = i2c.NewBitBang(esp32s3.GPIO(47), esp32s3.GPIO(48), &Clock, 100000)
var internalI2CPort = i2c.DefinePort(&internalI2CController, &Clock)
var internalI2CBus = i2c.New(internalI2CPort)
var paperPMIC = m5pm1.New(internalI2CBus)
var paperIOExpander = m5ioe1.New(internalI2CBus)
var epdChipSelect = esp32s3.GPIO(16)

// Power controls only the PaperMono-Lite display and touch power domains.
// Other PMIC and expander outputs remain untouched.
var Power = newPowerDevice(paperPMIC, paperIOExpander, epdChipSelect, &Clock)

type powerPMIC interface {
	Identify() (uint16, error)
	Initialize() error
	ConfigurePWM(m5pm1.Pin, uint16) error
	SetPWMDuty(m5pm1.Pin, uint16) error
}

type powerIOExpander interface {
	Identify() (uint16, error)
	Initialize() error
	ConfigureOutput(m5ioe1.Pin, bool) error
	SetOutput(m5ioe1.Pin, bool) error
	Output(m5ioe1.Pin) (bool, error)
}

type powerOutputPin interface {
	Set(bool)
	Configure(gpio.Config) error
}

type powerDelay interface {
	DelayMilliseconds(uint32)
}

// PowerIdentity contains the IDs read by a non-mutating power-bus probe.
type PowerIdentity struct {
	PMIC       uint16
	IOExpander uint16
}

// PowerDevice sequences the PaperMono-Lite display and touch domains.
type PowerDevice struct {
	pmic                 powerPMIC
	expander             powerIOExpander
	chipSelect           powerOutputPin
	delay                powerDelay
	prepared             bool
	active               bool
	frontLightConfigured bool
}

func newPowerDevice(pmic powerPMIC, expander powerIOExpander, chipSelect powerOutputPin, delay powerDelay) *PowerDevice {
	return &PowerDevice{pmic: pmic, expander: expander, chipSelect: chipSelect, delay: delay}
}

// Probe validates the M5PM1 identity and reads the M5IOE1 UID without changing
// any register or GPIO.
func (p *PowerDevice) Probe() (PowerIdentity, error) {
	pmicID, err := p.pmic.Identify()
	if err != nil {
		return PowerIdentity{PMIC: pmicID}, err
	}
	expanderID, err := p.expander.Identify()
	if err != nil {
		return PowerIdentity{PMIC: pmicID}, err
	}
	return PowerIdentity{PMIC: pmicID, IOExpander: expanderID}, nil
}

// EnableDisplayAndTouch powers the EPD and touch domains with both resets held
// low, waits eight milliseconds, then releases reset. EPD chip select is driven
// high before the rail is enabled, and every resulting latch is read back.
func (p *PowerDevice) EnableDisplayAndTouch() error {
	if p.active {
		return nil
	}
	if err := p.prepare(); err != nil {
		return err
	}
	// Establish inactive persistent latches before enabling any output driver.
	if err := p.configureInactive(); err != nil {
		p.rollback()
		return err
	}
	if err := p.expander.SetOutput(m5ioe1.Pin3, true); err != nil {
		p.rollback()
		return err
	}
	if err := p.expander.SetOutput(m5ioe1.Pin13, true); err != nil {
		p.rollback()
		return err
	}
	p.delay.DelayMilliseconds(8)
	if err := p.expander.SetOutput(m5ioe1.Pin5, true); err != nil {
		p.rollback()
		return err
	}
	if err := p.expander.SetOutput(m5ioe1.Pin6, true); err != nil {
		p.rollback()
		return err
	}
	p.delay.DelayMilliseconds(2)
	for _, pin := range []m5ioe1.Pin{m5ioe1.Pin3, m5ioe1.Pin13, m5ioe1.Pin5, m5ioe1.Pin6} {
		if err := p.verify(pin, true); err != nil {
			p.rollback()
			return err
		}
	}
	p.active = true
	return nil
}

// ResetDisplay applies the SSD1677 hardware reset pulse while its power domain
// is active. It never changes touch or display power enables.
func (p *PowerDevice) ResetDisplay() error {
	if !p.active {
		return ErrPowerInactive
	}
	if err := p.expander.SetOutput(m5ioe1.Pin5, false); err != nil {
		return err
	}
	p.delay.DelayMilliseconds(10)
	if err := p.expander.SetOutput(m5ioe1.Pin5, true); err != nil {
		return err
	}
	p.delay.DelayMilliseconds(10)
	return p.verify(m5ioe1.Pin5, true)
}

// SetFrontLight sets the integrated front light from off (0) to full (255).
// PaperMono-Lite connects BL_FB to M5PM1 GPIO3/PWM0. The squared brightness
// curve matches M5Stack's board support and gives useful resolution at the dim
// end of the range.
func (p *PowerDevice) SetFrontLight(brightness uint8) error {
	if err := p.prepare(); err != nil {
		return err
	}
	if !p.frontLightConfigured {
		if err := p.pmic.ConfigurePWM(m5pm1.Pin3, 5000); err != nil {
			return err
		}
		p.frontLightConfigured = true
	}
	squared := uint32(brightness) * uint32(brightness)
	return p.pmic.SetPWMDuty(m5pm1.Pin3, uint16(squared>>4))
}

// DisableDisplayAndTouch asserts both resets before removing touch and EPD
// power. Every step is attempted even after an I2C error, and EPD chip select
// remains inactive. No SSD1677 command is issued by this Phase 2 operation.
func (p *PowerDevice) DisableDisplayAndTouch() error {
	wasPrepared := p.prepared
	if !p.prepared {
		if err := p.prepare(); err != nil {
			return err
		}
	}
	var first error
	if !wasPrepared {
		first = p.configureInactive()
	}
	p.chipSelect.Set(true)
	if p.frontLightConfigured {
		first = retainFirst(first, p.pmic.SetPWMDuty(m5pm1.Pin3, 0))
	}
	first = retainFirst(first, p.expander.SetOutput(m5ioe1.Pin5, false))
	first = retainFirst(first, p.expander.SetOutput(m5ioe1.Pin6, false))
	p.delay.DelayMilliseconds(2)
	first = retainFirst(first, p.expander.SetOutput(m5ioe1.Pin13, false))
	first = retainFirst(first, p.expander.SetOutput(m5ioe1.Pin3, false))
	for _, pin := range []m5ioe1.Pin{m5ioe1.Pin3, m5ioe1.Pin13, m5ioe1.Pin5, m5ioe1.Pin6} {
		first = retainFirst(first, p.verify(pin, false))
	}
	if first == nil {
		p.active = false
	}
	return first
}

func (p *PowerDevice) prepare() error {
	if p.prepared {
		return nil
	}
	if _, err := p.Probe(); err != nil {
		return err
	}
	p.chipSelect.Set(true)
	if err := p.chipSelect.Configure(gpio.Config{Direction: gpio.Output}); err != nil {
		return err
	}
	if err := p.pmic.Initialize(); err != nil {
		return err
	}
	if err := p.expander.Initialize(); err != nil {
		return err
	}
	p.prepared = true
	return nil
}

func (p *PowerDevice) configureInactive() error {
	var first error
	for _, pin := range []m5ioe1.Pin{m5ioe1.Pin5, m5ioe1.Pin6, m5ioe1.Pin3, m5ioe1.Pin13} {
		first = retainFirst(first, p.expander.ConfigureOutput(pin, false))
	}
	return first
}

func (p *PowerDevice) verify(pin m5ioe1.Pin, want bool) error {
	got, err := p.expander.Output(pin)
	if err != nil {
		return err
	}
	if got != want {
		return ErrPowerLatch
	}
	return nil
}

func (p *PowerDevice) rollback() { _ = p.DisableDisplayAndTouch() }

func retainFirst(first, next error) error {
	if first != nil {
		return first
	}
	return next
}

type powerError string

func (e powerError) Error() string { return string(e) }

// ErrPowerLatch reports that a display or touch power latch did not retain the
// requested level.
const ErrPowerLatch powerError = "PaperMono-Lite power latch verification failed"

// ErrPowerInactive reports a reset request made with the EPD rail disabled.
const ErrPowerInactive powerError = "PaperMono-Lite display power is inactive"
