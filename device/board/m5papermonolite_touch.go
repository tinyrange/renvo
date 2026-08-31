//go:build m5papermonolite

package board

import (
	"renvo.dev/device/esp32s3"
	"renvo.dev/device/gpio"
	"renvo.dev/device/input/ft6336g"
)

var touchInterruptPin = esp32s3.GPIO(4)
var paperTouchController = ft6336g.New(internalI2CBus.Device(ft6336g.Address))

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

// Read returns the first normalized contact. A high interrupt level is treated
// as a release without issuing an unnecessary I2C transaction.
func (touch *Touchscreen) Read() (point ft6336g.Point, pressed bool, err error) {
	if !touch.ready {
		return point, false, ErrTouchNotInitialized
	}
	if !touch.Pending() {
		return point, false, nil
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
