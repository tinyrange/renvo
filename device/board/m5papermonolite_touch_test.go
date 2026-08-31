//go:build m5papermonolite

package board

import (
	"errors"
	"testing"

	"renvo.dev/device/gpio"
	"renvo.dev/device/input/ft6336g"
)

type fakeTouchController struct {
	initializeCalls int
	readCalls       int
	identity        ft6336g.Identity
	report          ft6336g.Report
	err             error
}

func (controller *fakeTouchController) Initialize() (ft6336g.Identity, error) {
	controller.initializeCalls++
	return controller.identity, controller.err
}

func (controller *fakeTouchController) Read() (ft6336g.Report, error) {
	controller.readCalls++
	return controller.report, controller.err
}

type fakeTouchInterrupt struct {
	high           bool
	configureCalls int
	config         gpio.Config
}

func (interrupt *fakeTouchInterrupt) Configure(config gpio.Config) error {
	interrupt.configureCalls++
	interrupt.config = config
	return nil
}

func (interrupt *fakeTouchInterrupt) Get() bool { return interrupt.high }

type fakeTouchDisplay struct {
	enableCalls   int
	shutdownCalls int
}

func (display *fakeTouchDisplay) Enable() error {
	display.enableCalls++
	return nil
}

func (display *fakeTouchDisplay) Shutdown() error {
	display.shutdownCalls++
	return nil
}

func TestTouchscreenInitializesPowerInterruptAndControllerOnce(t *testing.T) {
	controller := &fakeTouchController{identity: ft6336g.Identity{Vendor: 0x11}}
	interrupt := &fakeTouchInterrupt{high: true}
	display := &fakeTouchDisplay{}
	touch := newTouchscreen(controller, interrupt, display)
	identity, err := touch.Initialize()
	if err != nil || identity.Vendor != 0x11 {
		t.Fatalf("Initialize = %+v, %v", identity, err)
	}
	if _, err := touch.Initialize(); err != nil {
		t.Fatal(err)
	}
	if controller.initializeCalls != 1 || interrupt.configureCalls != 1 || display.enableCalls != 1 {
		t.Fatalf("controller=%d interrupt=%d power=%d", controller.initializeCalls, interrupt.configureCalls, display.enableCalls)
	}
	if interrupt.config.Direction != gpio.Input || interrupt.config.Pull != gpio.PullUp {
		t.Fatalf("interrupt config = %+v", interrupt.config)
	}
}

func TestTouchscreenPollsCoordinatesIndependentOfInterruptLevel(t *testing.T) {
	controller := &fakeTouchController{identity: ft6336g.Identity{Vendor: 1}}
	controller.report.Count = 1
	controller.report.Points[0] = ft6336g.Point{X: 12, Y: 34}
	interrupt := &fakeTouchInterrupt{high: true}
	touch := newTouchscreen(controller, interrupt, &fakeTouchDisplay{})
	if _, err := touch.Initialize(); err != nil {
		t.Fatal(err)
	}
	point, pressed, err := touch.Read()
	if err != nil || !pressed || point.X != 12 || point.Y != 34 || controller.readCalls != 1 {
		t.Fatalf("pressed Read = %+v, %v, %v calls=%d", point, pressed, err, controller.readCalls)
	}
	controller.report.Count = 0
	if _, pressed, err := touch.Read(); err != nil || pressed || controller.readCalls != 2 {
		t.Fatalf("released Read = %v, %v calls=%d", pressed, err, controller.readCalls)
	}
}

func TestTouchscreenInitializationFailurePowersDown(t *testing.T) {
	injected := errors.New("injected touch failure")
	display := &fakeTouchDisplay{}
	touch := newTouchscreen(&fakeTouchController{err: injected}, &fakeTouchInterrupt{}, display)
	if _, err := touch.Initialize(); err != injected {
		t.Fatalf("Initialize error = %v", err)
	}
	if display.shutdownCalls != 1 || touch.ready {
		t.Fatalf("shutdown=%d ready=%v", display.shutdownCalls, touch.ready)
	}
}

func TestTouchscreenRejectsReadBeforeInitialization(t *testing.T) {
	touch := newTouchscreen(&fakeTouchController{}, &fakeTouchInterrupt{}, &fakeTouchDisplay{})
	if _, _, err := touch.Read(); err != ErrTouchNotInitialized {
		t.Fatalf("Read error = %v", err)
	}
}
