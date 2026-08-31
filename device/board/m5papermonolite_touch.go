//go:build m5papermonolite

package board

import (
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
	"renvo.dev/device/i2c"
	"renvo.dev/device/input/ft6336g"
)

var touchInterruptPin = esp32s3.GPIO(4)
var touchI2CController = i2c.NewBitBang(esp32s3.GPIO(47), esp32s3.GPIO(48), &Clock, 400000)
var touchI2CPort = i2c.DefinePort(&touchI2CController, &Clock)
var touchI2CBus = i2c.New(touchI2CPort)
var paperTouchController = ft6336g.New(touchI2CBus.Device(ft6336g.Address))

type touchController interface {
	Initialize() (ft6336g.Identity, error)
	Read() (ft6336g.Report, error)
}

type touchInterrupt interface {
	Configure(gpio.Config) error
	Get() bool
}

type touchDisplayPower interface {
	Enable() error
	Shutdown() error
}

// Touchscreen owns PaperMono-Lite touch power and GPIO4 interrupt policy.
type Touchscreen struct {
	controller touchController
	interrupt  touchInterrupt
	display    touchDisplayPower
	identity   ft6336g.Identity
	ready      bool
}

// Touch is the PaperMono-Lite FT6336G touch controller.
var Touch = newTouchscreen(paperTouchController, touchInterruptPin, Display)

func newTouchscreen(controller touchController, interrupt touchInterrupt, display touchDisplayPower) *Touchscreen {
	return &Touchscreen{controller: controller, interrupt: interrupt, display: display}
}

// Initialize enables the shared display/touch power domains, configures the
// active-low GPIO4 interrupt with a pull-up, and validates the FT6336G.
func (touch *Touchscreen) Initialize() (ft6336g.Identity, error) {
	if touch.ready {
		return touch.identity, nil
	}
	if err := touch.interrupt.Configure(gpio.Config{Direction: gpio.Input, Pull: gpio.PullUp}); err != nil {
		return ft6336g.Identity{}, err
	}
	if err := touch.display.Enable(); err != nil {
		return ft6336g.Identity{}, err
	}
	identity, err := touch.controller.Initialize()
	if err != nil {
		_ = touch.display.Shutdown()
		return identity, err
	}
	touch.identity = identity
	touch.ready = true
	return identity, nil
}

// Pending reports the active-low GPIO4 touch interrupt level.
func (touch *Touchscreen) Pending() bool { return !touch.interrupt.Get() }

// Read polls the controller and returns the first normalized contact. The
// FT6336G is configured in polling mode, so GPIO4 is only an activity hint and
// must not gate coordinate reads while a contact is moving.
func (touch *Touchscreen) Read() (point ft6336g.Point, pressed bool, err error) {
	if !touch.ready {
		return point, false, ErrTouchNotInitialized
	}
	report, err := touch.controller.Read()
	if err != nil {
		return point, false, err
	}
	if report.Count == 0 {
		return point, false, nil
	}
	return report.Points[0], true, nil
}

type touchError string

func (err touchError) Error() string { return string(err) }

// ErrTouchNotInitialized reports a touch read before Initialize.
const ErrTouchNotInitialized touchError = "PaperMono-Lite touch is not initialized"
