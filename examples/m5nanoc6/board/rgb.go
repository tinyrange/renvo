package board

const (
	ioMuxGPIO19        = uintptr(0x60090050)
	ioMuxGPIO20        = uintptr(0x60090054)
	gpio19OutSelect    = uintptr(0x600915a0)
	gpio20OutSelect    = uintptr(0x600915a4)
	rmtConfig          = uintptr(0x60006010)
	rmtInterruptRaw    = uintptr(0x60006038)
	rmtInterruptEnable = uintptr(0x60006040)
	rmtInterruptClear  = uintptr(0x60006044)
	rmtSystemConfig    = uintptr(0x60006068)
	rmtMemory          = uintptr(0x60006400)
	pcrRMTConfig       = uintptr(0x6009602c)
	pcrRMTClockConfig  = uintptr(0x60096030)

	rgbPower = uint32(1 << 19)
	rgbData  = uint32(1 << 20)

	rmtOutputSignal = uint32(71)
	rmtClockEnable  = uint32(1)
	rmtReset        = uint32(1 << 1)
	rmtClock80MHz   = uint32(1 << 20)
	rmtSourceEnable = uint32(1 << 22)
	rmtDirectMemory = uint32(1)
	rmtIdleLow      = uint32(1 << 6)
	rmtDivider10MHz = uint32(8 << 8)
	rmtMemoryBlock  = uint32(1 << 16)
	rmtUpdate       = uint32(1 << 24)
	rmtStart        = uint32(1)
	rmtResetRead    = uint32(1 << 1)
	rmtResetMemory  = uint32(1 << 2)
	rmtTransmitDone = uint32(1)
)

// ConfigureRGB powers the NanoC6's WS2812 and connects RMT channel 0 to its
// GPIO20 data input. The RMT runs at 10 MHz, giving each timing tick 0.1 us.
func ConfigureRGB() {
	store32(ioMuxGPIO19,
		(load32(ioMuxGPIO19)&^gpioFunctionMask)|gpioFunction)
	store32(gpio19OutSelect, gpioOutputSignal)
	store32(gpioOutSet, rgbPower)
	store32(gpioEnableSet, rgbPower)

	store32(ioMuxGPIO20,
		(load32(ioMuxGPIO20)&^gpioFunctionMask)|gpioFunction)
	store32(gpio20OutSelect, rmtOutputSignal)
	store32(gpioEnableSet, rgbData)
	resetRMT()
}

func resetRMT() {
	clock := load32(pcrRMTConfig) | rmtClockEnable
	store32(pcrRMTConfig, clock|rmtReset)
	store32(pcrRMTConfig, clock&^rmtReset)
	store32(pcrRMTClockConfig, rmtClock80MHz|rmtSourceEnable)
	store32(rmtSystemConfig, rmtDirectMemory)
	store32(rmtInterruptEnable, 0)
	store32(rmtInterruptClear, uint32(0xffffffff))
	store32(rmtConfig, rmtIdleLow|rmtDivider10MHz|rmtMemoryBlock)
}

func rmtSymbol(one bool) uint32 {
	high := uint32(3)
	low := uint32(9)
	if one {
		high = 9
		low = 3
	}
	return high | uint32(1<<15) | low<<16
}

// SetRGB sends one WS2812 color in its native GRB, most-significant-bit-first
// wire order. Red, green, and blue remain the friendlier caller-facing order.
func SetRGB(red uint8, green uint8, blue uint8) {
	color := uint32(green)<<16 | uint32(red)<<8 | uint32(blue)
	bit := uint32(1 << 23)
	address := rmtMemory
	for bit != 0 {
		store32(address, rmtSymbol(color&bit != 0))
		address = address + 4
		bit = bit >> 1
	}
	store32(address, 0)

	config := rmtIdleLow | rmtDivider10MHz | rmtMemoryBlock
	store32(rmtInterruptClear, rmtTransmitDone)
	store32(rmtConfig, config|rmtResetRead)
	store32(rmtConfig, config)
	store32(rmtConfig, config|rmtResetMemory)
	store32(rmtConfig, config)
	store32(rmtConfig, config|rmtUpdate)
	store32(rmtConfig, config|rmtStart)
	for load32(rmtInterruptRaw)&rmtTransmitDone == 0 {
	}
	store32(rmtInterruptClear, rmtTransmitDone)
}
