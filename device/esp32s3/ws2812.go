package esp32s3

import "renvo.dev/device/mmio"

const (
	rmtConfig          = uintptr(0x60016020)
	rmtInterruptRaw    = uintptr(0x60016070)
	rmtInterruptEnable = uintptr(0x60016078)
	rmtInterruptClear  = uintptr(0x6001607c)
	rmtSystemConfig    = uintptr(0x600160c0)
	rmtClockReset      = uintptr(0x600160c8)
	rmtMemory          = uintptr(0x60016800)
	systemClock0       = uintptr(0x600c0018)
	systemReset0       = uintptr(0x600c0020)

	rmtOutputSignal = uint32(81)
	rmtClockEnable  = uint32(1 << 9)
	rmtReset        = uint32(1 << 9)
	rmtDirectMemory = uint32(1)
	rmtMemoryPower  = uint32(1 << 3)
	rmtSourceActive = uint32(1 << 26)
	rmtSourceXTAL   = uint32(3 << 24)
	rmtIdleLow      = uint32(1 << 6)
	rmtDivider10MHz = uint32(4 << 8)
	rmtMemoryBlock  = uint32(1 << 16)
	rmtWrap         = uint32(1 << 4)
	rmtUpdate       = uint32(1 << 24)
	rmtStart        = uint32(1)
	rmtResetRead    = uint32(1 << 1)
	rmtResetMemory  = uint32(1 << 2)
	rmtTransmitDone = uint32(1)
)

// WS2812 is one ESP32-S3 RMT-backed RGB pixel with an optional power pin.
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
	if p.power != nil {
		p.power.Set(true)
		_ = p.power.ConfigureOutputSignal(gpioOutputSignal)
	}
	p.data.Set(false)
	_ = p.data.ConfigureOutputSignal(rmtOutputSignal)

	mmio.Store32(systemClock0, mmio.Load32(systemClock0)|rmtClockEnable)
	reset := mmio.Load32(systemReset0)
	mmio.Store32(systemReset0, reset|rmtReset)
	mmio.Store32(systemReset0, reset&^rmtReset)
	mmio.Store32(rmtSystemConfig, rmtDirectMemory|rmtMemoryPower|rmtSourceXTAL|rmtSourceActive)
	mmio.Store32(rmtInterruptEnable, 0)
	mmio.Store32(rmtInterruptClear, uint32(0xffffffff))
	mmio.Store32(rmtClockReset, 1)
	mmio.Store32(rmtConfig, rmtIdleLow|rmtDivider10MHz|rmtMemoryBlock|rmtWrap)
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

	config := rmtIdleLow | rmtDivider10MHz | rmtMemoryBlock | rmtWrap
	mmio.Store32(rmtInterruptClear, rmtTransmitDone)
	mmio.Store32(rmtConfig, config|rmtResetRead)
	mmio.Store32(rmtConfig, config)
	mmio.Store32(rmtConfig, config|rmtResetMemory)
	mmio.Store32(rmtConfig, config)
	mmio.Store32(rmtConfig, config|rmtUpdate)
	mmio.Store32(rmtConfig, config|rmtStart)
	timer := SystemTimer{}
	started := timer.Ticks()
	for mmio.Load32(rmtInterruptRaw)&rmtTransmitDone == 0 && timer.Ticks()-started < 160000 {
	}
	mmio.Store32(rmtInterruptClear, rmtTransmitDone)
}
