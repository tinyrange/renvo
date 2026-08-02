// Package board exposes the small amount of StickS3 hardware used by the
// bring-up oracle without depending on ESP-IDF.
package board

import "unsafe"

const (
	ioMuxGPIO38      = uintptr(0x6000909c)
	ioMuxGPIO11      = uintptr(0x60009030)
	ioMuxGPIO12      = uintptr(0x60009034)
	ioMuxGPIO39      = uintptr(0x600090a0)
	ioMuxGPIO40      = uintptr(0x600090a4)
	ioMuxGPIO41      = uintptr(0x600090a8)
	ioMuxGPIO45      = uintptr(0x600090b8)
	ioMuxGPIO47      = uintptr(0x600090c0)
	ioMuxGPIO48      = uintptr(0x600090c4)
	ioMuxGPIO21      = uintptr(0x60009058)
	gpioOutSet       = uintptr(0x60004008)
	gpioOutClear     = uintptr(0x6000400c)
	gpioOut1         = uintptr(0x60004010)
	gpioOut1Set      = uintptr(0x60004014)
	gpioOut1Clear    = uintptr(0x60004018)
	gpioEnableSet    = uintptr(0x60004024)
	gpioEnableClear  = uintptr(0x60004028)
	gpioEnable1Set   = uintptr(0x60004030)
	gpioEnable1Clear = uintptr(0x60004034)
	gpioInput1       = uintptr(0x60004040)
	gpioInput        = uintptr(0x6000403c)
	gpio38OutSelect  = uintptr(0x600045ec)
	gpio39OutSelect  = uintptr(0x600045f0)
	gpio40OutSelect  = uintptr(0x600045f4)
	gpio41OutSelect  = uintptr(0x600045f8)
	gpio45OutSelect  = uintptr(0x60004608)
	gpio47OutSelect  = uintptr(0x60004610)
	gpio48OutSelect  = uintptr(0x60004614)
	gpio21OutSelect  = uintptr(0x600045a8)

	gpioFunctionMask = uint32(7 << 12)
	gpioFunction     = uint32(1 << 12)
	gpioInputEnable  = uint32(1 << 9)
	gpioPullUp       = uint32(1 << 8)
	gpioPullDown     = uint32(1 << 7)
	gpioOutputSignal = uint32(256)
)

func load32(address uintptr) uint32 {
	return *(*uint32)(unsafe.Pointer(address))
}

func store32(address uintptr, value uint32) {
	*(*uint32)(unsafe.Pointer(address)) = value
}

func pinBit(pin int) uint32 {
	if pin >= 32 {
		pin -= 32
	}
	return uint32(1) << uint(pin)
}

func pinOutputSelect(pin int) uintptr {
	switch pin {
	case 21:
		return gpio21OutSelect
	case 38:
		return gpio38OutSelect
	case 39:
		return gpio39OutSelect
	case 40:
		return gpio40OutSelect
	case 41:
		return gpio41OutSelect
	case 45:
		return gpio45OutSelect
	case 47:
		return gpio47OutSelect
	default:
		return gpio48OutSelect
	}
}

func pinIOMux(pin int) uintptr {
	switch pin {
	case 11:
		return ioMuxGPIO11
	case 12:
		return ioMuxGPIO12
	case 21:
		return ioMuxGPIO21
	case 38:
		return ioMuxGPIO38
	case 39:
		return ioMuxGPIO39
	case 40:
		return ioMuxGPIO40
	case 41:
		return ioMuxGPIO41
	case 45:
		return ioMuxGPIO45
	case 47:
		return ioMuxGPIO47
	default:
		return ioMuxGPIO48
	}
}

// ConfigureButton leaves a StickS3 button pin input-only. The board supplies
// the pull-up; pressing a button connects its GPIO to ground.
func ConfigureButton(pin int) {
	value := load32(pinIOMux(pin)) &^ (gpioFunctionMask | gpioPullUp | gpioPullDown)
	value |= gpioFunction | gpioInputEnable
	store32(pinIOMux(pin), value)
	enableGPIO(pin, false)
}

func ButtonPressed(pin int) bool {
	return load32(gpioInput)&pinBit(pin) == 0
}

func configureGPIO(pin int, input bool) {
	value := load32(pinIOMux(pin)) &^ gpioFunctionMask
	value |= gpioFunction
	if input {
		value |= gpioInputEnable | gpioPullUp
	}
	store32(pinIOMux(pin), value)
	store32(pinOutputSelect(pin), gpioOutputSignal)
}

func connectGPIOOutput(pin int, signal uint32) {
	store32(pinOutputSelect(pin), signal)
}

func enableGPIO(pin int, enable bool) {
	bit := pinBit(pin)
	if pin < 32 {
		if enable {
			store32(gpioEnableSet, bit)
		} else {
			store32(gpioEnableClear, bit)
		}
		return
	}
	if enable {
		store32(gpioEnable1Set, bit)
	} else {
		store32(gpioEnable1Clear, bit)
	}
}

func setGPIO(pin int, high bool) {
	bit := pinBit(pin)
	if pin < 32 {
		if high {
			store32(gpioOutSet, bit)
		} else {
			store32(gpioOutClear, bit)
		}
		return
	}
	if high {
		store32(gpioOut1Set, bit)
	} else {
		store32(gpioOut1Clear, bit)
	}
}

// ConfigureBacklight connects GPIO38 to the LCD backlight and starts dark.
func ConfigureBacklight() {
	configureGPIO(38, false)
	setGPIO(38, false)
	enableGPIO(38, true)
}

func SetBacklight(on bool) {
	setGPIO(38, on)
}

// Delay performs observable MMIO reads so the compiler cannot remove it.
func Delay(iterations int) {
	for count := 0; count < iterations; count++ {
		_ = load32(gpioOut1)
	}
}
