package esp32c6

import (
	"renvo.dev/device/gpio"
	"renvo.dev/device/internal/esprmt"
	"renvo.dev/device/mmio"
	"renvo.dev/device/ws2812"
)

const (
	rmtConfig          = uintptr(0x60006010)
	rmtInterruptRaw    = uintptr(0x60006038)
	rmtInterruptEnable = uintptr(0x60006040)
	rmtInterruptClear  = uintptr(0x60006044)
	rmtTransmitLimit   = uintptr(0x60006058)
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
	rmtMemoryBlocks = uint32(4 << 16)
	rmtWrap         = uint32(1 << 4)

	rmtMemoryWords = 4 * 48
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

// WS2812Transmitter selects the ESP32-C6 RMT peripheral for pixels attached to
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

	clock := mmio.Load32(pcrRMTConfig) | rmtClockEnable
	mmio.Store32(pcrRMTConfig, clock|rmtReset)
	mmio.Store32(pcrRMTConfig, clock&^rmtReset)
	mmio.Store32(pcrRMTClockConfig, rmtClock80MHz|rmtSourceEnable)
	mmio.Store32(rmtSystemConfig, rmtDirectMemory)
	p.sender.Initialize()
	p.initialized = true
}

func (p *ws2812RMT) Transmit(data []byte) bool {
	p.initialize()
	timer := SystemTimer{}
	return p.sender.Transmit(data, &timer)
}
