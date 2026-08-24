package esp32s3

import (
	"renvo.dev/device/gpio"
	"renvo.dev/device/internal/esprmt"
	"renvo.dev/device/mmio"
	"renvo.dev/device/ws2812"
)

const (
	rmtConfig          = uintptr(0x60016020)
	rmtInterruptRaw    = uintptr(0x60016070)
	rmtInterruptEnable = uintptr(0x60016078)
	rmtInterruptClear  = uintptr(0x6001607c)
	rmtTransmitLimit   = uintptr(0x600160a0)
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
	rmtMemoryBlocks = uint32(8 << 16)
	rmtWrap         = uint32(1 << 4)

	rmtMemoryWords = 8 * 48
)

// RGB is one red, green and blue addressable-LED value.
type RGB = ws2812.RGB

// WS2812 is the portable WS2812-compatible strip driver.
type WS2812 = ws2812.Strip

type ws2812RMT struct {
	data        *Pin
	power       gpio.Pin
	initialized bool
	sender      esprmt.Sender
}

// WS2812Transmitter selects the ESP32-S3 RMT peripheral for pixels attached to
// this pin. Portable code reaches it through ws2812.Output.
func (p *Pin) WS2812Transmitter(power gpio.Pin) ws2812.Transmitter {
	return newWS2812Transmitter(p, power)
}

func newWS2812Transmitter(data *Pin, power gpio.Pin) ws2812.Transmitter {
	transport := &ws2812RMT{data: data, power: power}
	transport.sender = esprmt.New(esprmt.Config{
		ConfigAddress:          rmtConfig,
		InterruptRawAddress:    rmtInterruptRaw,
		InterruptEnableAddress: rmtInterruptEnable,
		InterruptClearAddress:  rmtInterruptClear,
		TransmitLimitAddress:   rmtTransmitLimit,
		MemoryAddress:          rmtMemory,
		BaseConfig:             rmtIdleLow | rmtDivider10MHz | rmtMemoryBlocks | rmtWrap,
		MemoryWords:            rmtMemoryWords,
	})
	return transport
}

// NewWS2812 describes a pixel or strip routed through RMT channel zero.
// Deprecated: use ws2812.New with a board-provided output capability.
func NewWS2812(data, power *Pin) WS2812 {
	return ws2812.New(data, power)
}

func (p *ws2812RMT) initialize() {
	if p.initialized {
		return
	}
	if p.power != nil {
		p.power.Set(true)
		_ = p.power.Configure(gpio.Config{Direction: gpio.Output})
	}
	p.data.Set(false)
	_ = p.data.ConfigureOutputSignal(rmtOutputSignal)

	mmio.Store32(systemClock0, mmio.Load32(systemClock0)|rmtClockEnable)
	reset := mmio.Load32(systemReset0)
	mmio.Store32(systemReset0, reset|rmtReset)
	mmio.Store32(systemReset0, reset&^rmtReset)
	mmio.Store32(rmtSystemConfig, rmtDirectMemory|rmtMemoryPower|rmtSourceXTAL|rmtSourceActive)
	mmio.Store32(rmtClockReset, 1)
	p.sender.Initialize()
	p.initialized = true
}

func (p *ws2812RMT) Transmit(data []byte) bool {
	p.initialize()
	timer := SystemTimer{}
	return p.sender.Transmit(data, &timer)
}
