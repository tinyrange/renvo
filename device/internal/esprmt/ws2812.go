// Package esprmt implements the ESP RMT transmission shared by the supported
// microcontrollers. Chip packages provide only their register map and clocks.
package esprmt

import (
	"renvo.dev/device/clock"
	"renvo.dev/device/mmio"
)

const (
	start         = uint32(1)
	resetRead     = uint32(1 << 1)
	resetMemory   = uint32(1 << 2)
	stop          = uint32(1 << 7)
	update        = uint32(1 << 24)
	transmitDone  = uint32(1)
	transmitError = uint32(1 << 4)
	transmitHalf  = uint32(1 << 8)
)

// Config describes channel-zero registers and memory configured by a chip
// package. BaseConfig must include the clock divider, memory size, idle level,
// and transmit-wrap setting.
type Config struct {
	ConfigAddress          uintptr
	InterruptRawAddress    uintptr
	InterruptEnableAddress uintptr
	InterruptClearAddress  uintptr
	TransmitLimitAddress   uintptr
	MemoryAddress          uintptr
	BaseConfig             uint32
	MemoryWords            int
}

// Sender converts WS2812 bytes to 10 MHz RMT symbols and transmits them using
// channel-zero ping-pong memory.
type Sender struct {
	config  Config
	symbols []uint32
}

// New returns an RMT sender using config.
func New(config Config) Sender {
	return Sender{config: config}
}

// Initialize resets the channel registers after the chip package has enabled
// the peripheral clock.
func (s *Sender) Initialize() {
	mmio.Store32(s.config.InterruptEnableAddress, 0)
	mmio.Store32(s.config.InterruptClearAddress, uint32(0xffffffff))
	mmio.Store32(s.config.ConfigAddress, s.config.BaseConfig)
}

func symbol(one bool) uint32 {
	high, low := uint32(3), uint32(9)
	if one {
		high, low = 9, 3
	}
	return high | uint32(1<<15) | low<<16
}

func encode(data []byte, reuse []uint32) []uint32 {
	required := len(data)*8 + 1
	if cap(reuse) < required {
		reuse = make([]uint32, required)
	} else {
		reuse = reuse[:required]
	}
	at := 0
	for i := 0; i < len(data); i++ {
		value := data[i]
		for bit := uint8(0); bit < 8; bit++ {
			reuse[at] = symbol(value&(uint8(1)<<(7-bit)) != 0)
			at++
		}
	}
	reuse[at] = 0
	return reuse
}

func fill(address uintptr, count int, symbols []uint32, offset int) (int, bool) {
	for i := 0; i < count; i++ {
		value := uint32(0)
		if offset < len(symbols) {
			value = symbols[offset]
			offset++
		}
		mmio.Store32(address+uintptr(i*4), value)
	}
	return offset, offset == len(symbols)
}

// Transmit sends a complete WS2812 byte stream. The stream is pre-encoded so
// live RMT refills only copy words into peripheral memory.
func (s *Sender) Transmit(data []byte, timer clock.Source) bool {
	s.symbols = encode(data, s.symbols)
	config := s.config
	halfWords := config.MemoryWords / 2
	offset, complete := fill(config.MemoryAddress, config.MemoryWords, s.symbols, 0)
	interrupts := transmitDone | transmitError | transmitHalf

	mmio.Store32(config.InterruptClearAddress, interrupts)
	mmio.Store32(config.TransmitLimitAddress, uint32(halfWords))
	// Raw status is polled directly, so no CPU interrupt is required.
	mmio.Store32(config.InterruptEnableAddress, 0)
	mmio.Store32(config.ConfigAddress, config.BaseConfig|resetRead)
	mmio.Store32(config.ConfigAddress, config.BaseConfig)
	mmio.Store32(config.ConfigAddress, config.BaseConfig|resetMemory)
	mmio.Store32(config.ConfigAddress, config.BaseConfig)
	started := timer.Ticks()
	mmio.Store32(config.ConfigAddress, config.BaseConfig|update)
	mmio.Store32(config.ConfigAddress, config.BaseConfig|start)

	// Allow 10 ms of fixed overhead plus twice the nominal wire duration.
	timeout := timer.TicksPerSecond()/100 +
		uint32(len(s.symbols))*timer.TicksPerSecond()/500000
	refill := config.MemoryAddress
	for timer.Ticks()-started < timeout {
		status := mmio.Load32(config.InterruptRawAddress)
		if status&transmitDone != 0 {
			mmio.Store32(config.InterruptClearAddress, interrupts)
			return true
		}
		if status&transmitError != 0 {
			mmio.Store32(config.ConfigAddress, config.BaseConfig|stop)
			mmio.Store32(config.InterruptClearAddress, interrupts)
			return false
		}
		if status&transmitHalf != 0 {
			mmio.Store32(config.InterruptClearAddress, transmitHalf)
			if !complete {
				offset, complete = fill(refill, halfWords, s.symbols, offset)
			}
			if refill == config.MemoryAddress {
				refill += uintptr(halfWords * 4)
			} else {
				refill = config.MemoryAddress
			}
		}
	}
	mmio.Store32(config.ConfigAddress, config.BaseConfig|stop)
	mmio.Store32(config.InterruptClearAddress, interrupts)
	return false
}
