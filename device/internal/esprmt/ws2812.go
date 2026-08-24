// Package esprmt implements the ESP RMT transmission shared by the supported
// microcontrollers. Chip packages provide only their register map and clocks.
package esprmt

import (
	"unsafe"

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
	// At 10 MHz, two 250-tick low phases provide the 50 us reset interval used
	// by Espressif's reference LED-strip encoder.
	resetSymbol = uint32(250 | 250<<16)
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
	InterruptRefillAddress uintptr
	BaseConfig             uint32
	TransmitLimitConfig    uint32
	MemoryWords            int
}

// Sender converts WS2812 bytes to chip-timed RMT symbols and transmits them
// using channel-zero ping-pong memory.
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
	// The reference encoder terminates each byte stream with a 50 us low reset
	// symbol, followed by the zero-duration RMT EOF marker.
	required := len(data)*8 + 2
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
	reuse[at] = resetSymbol
	reuse[at+1] = 0
	return reuse
}

func pad(symbols []uint32, multiple int) []uint32 {
	required := (len(symbols) + multiple - 1) / multiple * multiple
	original := len(symbols)
	if cap(symbols) < required {
		padded := make([]uint32, required)
		copy(padded, symbols)
		symbols = padded
	} else {
		symbols = symbols[:required]
	}
	for i := original; i < required; i++ {
		symbols[i] = 0
	}
	return symbols
}

type burst16 struct {
	w0, w1, w2, w3     uint32
	w4, w5, w6, w7     uint32
	w8, w9, w10, w11   uint32
	w12, w13, w14, w15 uint32
}

type burst8 struct {
	w0, w1, w2, w3 uint32
	w4, w5, w6, w7 uint32
}

func copyBurst16(address, source uintptr) {
	destination := (*burst16)(unsafe.Pointer(address))
	values := (*burst16)(unsafe.Pointer(source))
	destination.w0 = values.w0
	destination.w1 = values.w1
	destination.w2 = values.w2
	destination.w3 = values.w3
	destination.w4 = values.w4
	destination.w5 = values.w5
	destination.w6 = values.w6
	destination.w7 = values.w7
	destination.w8 = values.w8
	destination.w9 = values.w9
	destination.w10 = values.w10
	destination.w11 = values.w11
	destination.w12 = values.w12
	destination.w13 = values.w13
	destination.w14 = values.w14
	destination.w15 = values.w15
}

func copyBurst8(address, source uintptr) {
	destination := (*burst8)(unsafe.Pointer(address))
	values := (*burst8)(unsafe.Pointer(source))
	destination.w0 = values.w0
	destination.w1 = values.w1
	destination.w2 = values.w2
	destination.w3 = values.w3
	destination.w4 = values.w4
	destination.w5 = values.w5
	destination.w6 = values.w6
	destination.w7 = values.w7
}

func fill(address uintptr, count int, symbols []uint32, offset int) int {
	// Transmit pads every frame before entering the deadline-sensitive refill
	// loop. Resolve and validate the source once, then copy a burst at a time.
	// Renvo intentionally does little loop optimization, so unrolling this hot
	// path is what keeps a C6 half-buffer refill below its 28.8 us
	// wire time.
	source := uintptr(unsafe.Pointer(&symbols[offset]))
	remaining := count
	for remaining >= 16 {
		copyBurst16(address, source)
		address += 64
		source += 64
		remaining -= 16
	}
	if remaining >= 8 {
		copyBurst8(address, source)
		address += 32
		source += 32
		remaining -= 8
	}
	for remaining > 0 {
		mmio.Store32(address, mmio.Load32(source))
		address += 4
		source += 4
		remaining--
	}
	return offset + count
}

func prepareInterruptRefill(config Config, symbols []uint32, offset, halfWords int) {
	mailbox := config.InterruptRefillAddress
	base := uintptr(unsafe.Pointer(&symbols[0]))
	mmio.Store32(mailbox, uint32(base+uintptr(offset*4)))
	mmio.Store32(mailbox+4, uint32(base+uintptr(len(symbols)*4)))
	mmio.Store32(mailbox+8, uint32(config.MemoryAddress))
	mmio.Store32(mailbox+12, uint32(halfWords*4))
	mmio.Store32(mailbox+16, uint32(config.InterruptClearAddress))
	mmio.Store32(mailbox+20, uint32(config.InterruptEnableAddress))
	mmio.Store32(mailbox+24, transmitHalf)
}

// Transmit sends a complete WS2812 byte stream. The stream is pre-encoded so
// live RMT refills only copy words into peripheral memory.
func (s *Sender) Transmit(data []byte, timer clock.Source) bool {
	s.symbols = encode(data, s.symbols)
	encodedWords := len(s.symbols)
	config := s.config
	// A whole-memory multiple guarantees that both the initial load and every
	// half-memory refill can use the branch-free copy path. The first encoded
	// zero-duration symbol remains the hardware EOF marker; padding is never put
	// on the wire.
	s.symbols = pad(s.symbols, config.MemoryWords)
	halfWords := config.MemoryWords / 2
	interrupts := transmitDone | transmitError | transmitHalf

	// This is the non-DMA form of ESP-IDF's transaction setup: reset both RMT
	// memory pointers, select a half-buffer threshold, fill the complete block,
	// then alternate the two halves on each threshold event.
	mmio.Store32(config.InterruptClearAddress, interrupts)
	mmio.Store32(config.TransmitLimitAddress, config.TransmitLimitConfig|uint32(halfWords))
	// The C6 interrupt path enables only the half-buffer threshold below. Done
	// and error remain polled here, as does the complete fallback path.
	mmio.Store32(config.InterruptEnableAddress, 0)
	mmio.Store32(config.ConfigAddress, config.BaseConfig|resetRead)
	mmio.Store32(config.ConfigAddress, config.BaseConfig)
	mmio.Store32(config.ConfigAddress, config.BaseConfig|resetMemory)
	mmio.Store32(config.ConfigAddress, config.BaseConfig)
	offset := fill(config.MemoryAddress, config.MemoryWords, s.symbols, 0)
	complete := offset == len(s.symbols)
	interruptRefill := config.InterruptRefillAddress != 0 && !complete
	if interruptRefill {
		prepareInterruptRefill(config, s.symbols, offset, halfWords)
		mmio.Store32(config.InterruptEnableAddress, transmitHalf)
	}
	started := timer.Ticks()
	mmio.Store32(config.ConfigAddress, config.BaseConfig|update)
	mmio.Store32(config.ConfigAddress, config.BaseConfig|start)

	// Allow 10 ms of fixed overhead plus twice the nominal wire duration.
	timeout := timer.TicksPerSecond()/100 +
		uint32(encodedWords)*timer.TicksPerSecond()/500000
	// memOffset and memEnd deliberately follow ESP-IDF's mem_off_bytes/mem_end
	// state machine. After the initial full-block fill, the first threshold
	// releases [0, half), and the next releases [half, full).
	memOffset := 0
	memEnd := halfWords
	polls := uint32(0)
	for {
		status := mmio.Load32(config.InterruptRawAddress)
		if status&transmitDone != 0 {
			mmio.Store32(config.InterruptEnableAddress, 0)
			mmio.Store32(config.InterruptClearAddress, interrupts)
			return true
		}
		if status&transmitError != 0 {
			mmio.Store32(config.InterruptEnableAddress, 0)
			mmio.Store32(config.ConfigAddress, config.BaseConfig|stop)
			mmio.Store32(config.ConfigAddress, config.BaseConfig|update)
			mmio.Store32(config.InterruptClearAddress, interrupts)
			return false
		}
		if status&transmitHalf != 0 && !interruptRefill {
			mmio.Store32(config.InterruptClearAddress, transmitHalf)
			if !complete {
				offset = fill(
					config.MemoryAddress+uintptr(memOffset*4),
					memEnd-memOffset,
					s.symbols,
					offset,
				)
				complete = offset == len(s.symbols)
			}
			memOffset = memEnd
			if memOffset == config.MemoryWords {
				memOffset = 0
			}
			memEnd = halfWords*3 - memEnd
		}
		// Interface dispatch and timer arithmetic are disproportionately costly
		// in the compact freestanding compiler. Keep them off the RMT service
		// path: raw status is polled continuously and the generous timeout only
		// needs occasional sampling.
		polls++
		if polls&255 == 0 && timer.Ticks()-started >= timeout {
			break
		}
	}
	mmio.Store32(config.InterruptEnableAddress, 0)
	mmio.Store32(config.ConfigAddress, config.BaseConfig|stop)
	mmio.Store32(config.ConfigAddress, config.BaseConfig|update)
	mmio.Store32(config.InterruptClearAddress, interrupts)
	return false
}
