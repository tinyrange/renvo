package board

const (
	gpioBase  = uintptr(0x500e0000)
	ioMuxBase = uintptr(0x500e1000)
)

func pinMask(pin int) uint32 {
	if pin < 32 {
		return uint32(1) << uint(pin)
	}
	return uint32(1) << uint(pin-32)
}

func gpioBankOffset(pin, low, high int) uintptr {
	if pin < 32 {
		return uintptr(low)
	}
	return uintptr(high)
}

func configureGPIO(pin int, input, pullUp bool) {
	mux := ioMuxBase + 4 + uintptr(pin*4)
	value := load32(mux)
	value &^= 7<<12 | 1<<9 | 1<<8 | 1<<7
	value |= 1 << 12
	if input {
		value |= 1 << 9
	}
	if pullUp {
		value |= 1 << 8
	}
	store32(mux, value)
	// Select the software GPIO output and its output-enable register.
	store32(gpioBase+0x558+uintptr(pin*4), 0x500)
}

func setGPIO(pin int, high bool) {
	offset := gpioBankOffset(pin, 0x08, 0x14)
	if !high {
		offset = gpioBankOffset(pin, 0x0c, 0x18)
	}
	store32(gpioBase+offset, pinMask(pin))
}

func enableGPIO(pin int, enabled bool) {
	offset := gpioBankOffset(pin, 0x24, 0x30)
	if !enabled {
		offset = gpioBankOffset(pin, 0x28, 0x34)
	}
	store32(gpioBase+offset, pinMask(pin))
}

func readGPIO(pin int) bool {
	offset := gpioBankOffset(pin, 0x3c, 0x40)
	return load32(gpioBase+offset)&pinMask(pin) != 0
}

func enableBacklight() {
	configureGPIO(22, false, false)
	setGPIO(22, true)
	enableGPIO(22, true)
}
