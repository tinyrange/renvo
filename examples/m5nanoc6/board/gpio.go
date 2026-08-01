// Package board exposes the small amount of M5NanoC6 hardware used by the
// bring-up examples. It intentionally uses the ESP32-C6 register interface
// directly so the examples do not depend on an SDK or runtime.
package board

import "unsafe"

const (
	ioMuxGPIO7      = uintptr(0x60090020)
	ioMuxGPIO9      = uintptr(0x60090028)
	gpioOut         = uintptr(0x60091004)
	gpioOutSet      = uintptr(0x60091008)
	gpioOutClear    = uintptr(0x6009100c)
	gpioEnableSet   = uintptr(0x60091024)
	gpioEnableClear = uintptr(0x60091028)
	gpioInput       = uintptr(0x6009103c)
	gpio7OutSelect  = uintptr(0x60091570)

	blueLED          = uint32(1 << 7)
	button           = uint32(1 << 9)
	gpioFunctionMask = uint32(7 << 12)
	gpioFunction     = uint32(1 << 12)
	gpioInputEnable  = uint32(1 << 9)
	gpioPullUp       = uint32(1 << 8)
	gpioPullDown     = uint32(1 << 7)
	gpioOutputSignal = uint32(128)
)

func load32(address uintptr) uint32 {
	return *(*uint32)(unsafe.Pointer(address))
}

func store32(address uintptr, value uint32) {
	*(*uint32)(unsafe.Pointer(address)) = value
}

// ConfigureBlueLED connects the GPIO matrix to the NanoC6 blue LED on GPIO7
// and starts with the output low.
func ConfigureBlueLED() {
	store32(ioMuxGPIO7, (load32(ioMuxGPIO7)&^gpioFunctionMask)|gpioFunction)
	store32(gpio7OutSelect, gpioOutputSignal)
	store32(gpioOutClear, blueLED)
	store32(gpioEnableSet, blueLED)
}

func SetBlueLED(on bool) {
	if on {
		store32(gpioOutSet, blueLED)
		return
	}
	store32(gpioOutClear, blueLED)
}

// ConfigureButton enables the active-low button input on GPIO9. GPIO9 is also
// an ESP32-C6 strapping pin, so the pull-up keeps it in its normal run state
// when the button is released.
func ConfigureButton() {
	value := load32(ioMuxGPIO9)
	value = value &^ (gpioFunctionMask | gpioPullDown)
	store32(ioMuxGPIO9, value|gpioFunction|gpioInputEnable|gpioPullUp)
}

func ButtonPressed() bool {
	return load32(gpioInput)&button == 0
}

// Delay performs volatile GPIO reads. Using an MMIO read rather than an empty
// loop keeps the delay observable to the compiler and deterministic in the
// emulator.
func Delay(iterations int) {
	for count := 0; count < iterations; count++ {
		_ = load32(gpioOut)
	}
}
