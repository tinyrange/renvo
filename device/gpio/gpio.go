// Package gpio defines board-independent digital IO capabilities and devices.
package gpio

// Direction selects whether a pin is sampled or driven.
type Direction uint8

const (
	// Input configures a pin for sampling.
	Input Direction = iota
	// Output configures a pin for driving.
	Output
)

// Pull selects the internal bias applied to an input.
type Pull uint8

const (
	// PullNone disables the pin's internal bias.
	PullNone Pull = iota
	// PullUp biases an undriven input high.
	PullUp
	// PullDown biases an undriven input low.
	PullDown
)

// Config describes ordinary GPIO operation. Chip-specific alternate functions
// remain in chip packages and are not exposed through this interface.
type Config struct {
	Direction Direction
	Pull      Pull
}

// Pin is the narrow capability needed by portable digital devices.
type Pin interface {
	Configure(Config) error
	Set(bool)
	Get() bool
}

// LED is a digital indicator with board-defined active polarity.
type LED struct {
	pin         Pin
	activeLow   bool
	initialized bool
}

// NewLED describes an LED attached to pin. No hardware is touched until its
// first operation, which keeps package-level board declarations side-effect
// free.
func NewLED(pin Pin, activeLow bool) LED {
	return LED{pin: pin, activeLow: activeLow}
}

func (l *LED) initialize() {
	if l.initialized {
		return
	}
	// Establish the inactive level before enabling the output so an active-low
	// LED does not flash during initialization.
	l.pin.Set(l.activeLow)
	_ = l.pin.Configure(Config{Direction: Output})
	l.initialized = true
}

// Set turns the LED on or off. It is a complete first operation: callers do
// not need a separate board initialization call.
func (l *LED) Set(on bool) {
	l.initialize()
	l.pin.Set(on != l.activeLow)
}

// Toggle changes the current visible state.
func (l *LED) Toggle() {
	l.initialize()
	l.pin.Set(!l.pin.Get())
}

// Button is a digital input with board-defined active polarity and pull.
type Button struct {
	pin         Pin
	pull        Pull
	activeLow   bool
	initialized bool
}

// NewButton describes a button attached to pin.
func NewButton(pin Pin, pull Pull, activeLow bool) Button {
	return Button{pin: pin, pull: pull, activeLow: activeLow}
}

// Pressed reports whether the button is active. Configuration is performed on
// the first call.
func (b *Button) Pressed() bool {
	if !b.initialized {
		_ = b.pin.Configure(Config{Direction: Input, Pull: b.pull})
		b.initialized = true
	}
	return b.pin.Get() != b.activeLow
}
