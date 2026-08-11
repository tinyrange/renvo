package esp32c6

import "renvo.dev/device/mmio"

const (
	rmtConfig          = uintptr(0x60006010)
	rmtInterruptRaw    = uintptr(0x60006038)
	rmtInterruptEnable = uintptr(0x60006040)
	rmtInterruptClear  = uintptr(0x60006044)
	rmtSystemConfig    = uintptr(0x60006068)
	rmtMemory          = uintptr(0x60006400)
	pcrRMTConfig       = uintptr(0x6009602c)
	pcrRMTClockConfig  = uintptr(0x60096030)

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

// WS2812 is one ESP32-C6 RMT-backed RGB pixel with an optional power pin.
type WS2812 struct {
	data        *Pin
	power       *Pin
	initialized bool
}

// NewWS2812 describes a pixel routed through RMT channel zero.
func NewWS2812(data, power *Pin) WS2812 { return WS2812{data: data, power: power} }

func (p *WS2812) initialize() {
	if p.initialized {
		return
	}
	p.power.Set(true)
	_ = p.power.ConfigureOutputSignal(gpioOutputSignal)
	p.data.Set(false)
	_ = p.data.ConfigureOutputSignal(rmtOutputSignal)

	clock := mmio.Load32(pcrRMTConfig) | rmtClockEnable
	mmio.Store32(pcrRMTConfig, clock|rmtReset)
	mmio.Store32(pcrRMTConfig, clock&^rmtReset)
	mmio.Store32(pcrRMTClockConfig, rmtClock80MHz|rmtSourceEnable)
	mmio.Store32(rmtSystemConfig, rmtDirectMemory)
	mmio.Store32(rmtInterruptEnable, 0)
	mmio.Store32(rmtInterruptClear, uint32(0xffffffff))
	mmio.Store32(rmtConfig, rmtIdleLow|rmtDivider10MHz|rmtMemoryBlock)
	p.initialized = true
}

func rmtSymbol(one bool) uint32 {
	high, low := uint32(3), uint32(9)
	if one {
		high, low = 9, 3
	}
	return high | uint32(1<<15) | low<<16
}

// Set emits red, green and blue in WS2812 GRB wire order.
func (p *WS2812) Set(red, green, blue uint8) {
	p.initialize()
	color := uint32(green)<<16 | uint32(red)<<8 | uint32(blue)
	bit, address := uint32(1<<23), rmtMemory
	for bit != 0 {
		mmio.Store32(address, rmtSymbol(color&bit != 0))
		address += 4
		bit >>= 1
	}
	mmio.Store32(address, 0)

	config := rmtIdleLow | rmtDivider10MHz | rmtMemoryBlock
	mmio.Store32(rmtInterruptClear, rmtTransmitDone)
	mmio.Store32(rmtConfig, config|rmtResetRead)
	mmio.Store32(rmtConfig, config)
	mmio.Store32(rmtConfig, config|rmtResetMemory)
	mmio.Store32(rmtConfig, config)
	mmio.Store32(rmtConfig, config|rmtUpdate)
	mmio.Store32(rmtConfig, config|rmtStart)
	for mmio.Load32(rmtInterruptRaw)&rmtTransmitDone == 0 {
	}
	mmio.Store32(rmtInterruptClear, rmtTransmitDone)
}
