package esp32c6

import "unsafe"

// RealtimeDedicatedGPIO returns the CPU-dedicated input CSR. The ESP32-C6
// target lowers this declaration to one csrr instruction.
//
// renvo:linkstatic esp32c6,realtime_dedicated_gpio
func RealtimeDedicatedGPIO() uint32 { return 0 }

// RealtimeCycle returns the fixed-frequency machine-cycle counter configured
// by the real-time owner. The ESP32-C6 target lowers it to one csrr instruction.
//
// renvo:linkstatic esp32c6,realtime_cycle
func RealtimeCycle() uint32 { return 0 }

// RealtimeStart120MHz fixes the HP clock at 120 MHz and starts the cycle
// counter consumed by the real-time primitives. Call it before taking over the
// USB pad; dynamic frequency scaling must remain disabled while it is active.
//
// renvo:linkstatic esp32c6,realtime_start_120mhz
func RealtimeStart120MHz() {}

// RealtimeEnterAt waits until cycle and enters a generated instruction block
// in internal SRAM. The block returns normally through ra. The target emits
// the wait loop and an instruction-cache fence inline at the call site.
//
// renvo:linkstatic esp32c6,realtime_enter_at
func RealtimeEnterAt(cycle uint32, entry uintptr) {}

// RealtimeCall publishes and calls an executable internal-SRAM block. It
// returns the block's a0 result.
//
// renvo:linkstatic esp32c6,realtime_call
func RealtimeCall(entry uintptr) uint32 { return 0 }

// RealtimeReceivePID waits for a full-speed packet and returns a validated PID.
// It stores the cycle at the first EOP SE0 sample through eopCycle, which must
// not be nil. The target emits fixed-ten-cycle SYNC/PID sampling directly at
// the call site. A zero PID indicates timeout or invalid synchronization/PID
// complement.
//
// renvo:linkstatic esp32c6,realtime_receive_pid
func RealtimeReceivePID(timeout uint32, eopCycle *uint32) byte { return 0 }

const fullSpeedWaveformWords = 800

// FullSpeedWaveform owns an unrolled PHY-test transmitter in executable
// internal SRAM. It must be statically allocated on ESP32-C6; placing generated
// instructions in a large Go stack frame is not supported. Packet preparation
// happens outside the packet-critical path; transmission contains only
// scheduled register stores and a return.
type FullSpeedWaveform struct {
	code  [fullSpeedWaveformWords]uint32
	words int
}

const (
	rvNOP         = uint32(0x00000013)
	rvReturn      = uint32(0x00008067)
	rvStoreT4AtT0 = uint32(0x01d2a023)
	rvStoreT5AtT0 = uint32(0x01e2a023)
	rvStoreT6AtT0 = uint32(0x01f2a023)
	rvStoreT1AtT0 = uint32(0x0062a023)
)

// BuildFullSpeed compiles an encoded USB packet into the calibrated 120 MHz
// PHY-test store schedule. states must include the conventional SE0, SE0, J
// EOP produced by device/usb/lowspeed's encoders.
func (w *FullSpeedWaveform) BuildFullSpeed(states []byte) bool {
	if len(states) < 3 || states[len(states)-3] != 0 ||
		states[len(states)-2] != 0 || states[len(states)-1] != 1 {
		w.words = 0
		return false
	}
	needed := 6 + len(states) - 3 + (len(states)-3)/5 + 14
	if needed > len(w.code) {
		w.words = 0
		return false
	}
	index := 0
	emit := func(instruction uint32) {
		w.code[index] = instruction
		index++
	}
	// t0 = TEST_REG, t4 = full-speed J, t5 = K, t6 = SE0, t1 = release.
	emit(0x6000f2b7) // lui t0, 0x6000f
	emit(0x01c28293) // addi t0, t0, 0x1c
	emit(0x00700e93) // li t4, 7
	emit(0x00b00f13) // li t5, 11
	emit(0x00300f93) // li t6, 3
	emit(0x00100313) // li t1, 1
	for symbol, line := range states[:len(states)-3] {
		if line == 1 {
			emit(rvStoreT4AtT0)
		} else if line == 2 {
			emit(rvStoreT5AtT0)
		} else {
			w.words = 0
			return false
		}
		// Empirically calibrated against the C6 peripheral bridge: four
		// 75 ns store cells followed by one 83 ns cell at 120 MHz.
		if symbol%5 == 4 {
			emit(rvNOP)
		}
	}
	emit(rvStoreT6AtT0)
	for delay := 0; delay < 10; delay++ {
		emit(rvNOP)
	}
	emit(rvStoreT4AtT0)
	emit(rvStoreT1AtT0)
	emit(rvReturn)
	for clear := index; clear < w.words; clear++ {
		w.code[clear] = 0
	}
	w.words = index
	return true
}

// Entry returns the executable internal-SRAM address of a built waveform.
func (w *FullSpeedWaveform) Entry() uintptr {
	if w.words == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&w.code[0]))
}

// RunAt enters a built waveform at an absolute machine-cycle deadline.
func (w *FullSpeedWaveform) RunAt(cycle uint32) bool {
	entry := w.Entry()
	if entry == 0 {
		return false
	}
	RealtimeEnterAt(cycle, entry)
	return true
}

const fullSpeedExchangeWords = 1800

// FullSpeedExchange owns an executable packet matcher and optional immediate
// response. It must be statically allocated on ESP32-C6. Both packets are
// prepared by the ordinary Go-side USB encoder; the generated block performs
// only deterministic line sampling, comparison, and PHY stores. This is
// intentionally a building block for endpoint dispatch rather than a USB
// policy layer.
type FullSpeedExchange struct {
	code  [fullSpeedExchangeWords]uint32
	words int
}

// FullSpeedPacket pins one encoder-produced line-state sequence at the
// executable-block boundary. Keep States unchanged while an exchange built
// from it is running.
type FullSpeedPacket struct {
	States []byte
}

func rvEncodeR(funct3, destination, left, right uint32) uint32 {
	return 0x33 | destination<<7 | funct3<<12 | left<<15 | right<<20
}

func rvEncodeI(funct3, destination, source uint32, immediate int) uint32 {
	return 0x13 | destination<<7 | funct3<<12 | source<<15 |
		uint32(immediate&0x0fff)<<20
}

func rvEncodeBranch(funct3, left, right uint32, displacement int) uint32 {
	value := uint32(displacement)
	return 0x63 | funct3<<12 | left<<15 | right<<20 |
		(value>>12&1)<<31 | (value>>5&0x3f)<<25 |
		(value>>1&0x0f)<<8 | (value>>11&1)<<7
}

func validFullSpeedPacket(packet *FullSpeedPacket) bool {
	if packet == nil {
		return false
	}
	states := packet.States
	if len(states) < 11 || states[0] != 2 || states[len(states)-3] != 0 ||
		states[len(states)-2] != 0 || states[len(states)-1] != 1 {
		return false
	}
	for _, line := range states[:len(states)-3] {
		if line != 1 && line != 2 {
			return false
		}
	}
	return true
}

// BuildFullSpeed compiles an exact host-packet match followed by an optional
// device response. A nil response makes the block a receive-only matcher. A
// mismatch returns zero; a match returns one after the response is released.
func (x *FullSpeedExchange) BuildFullSpeed(
	request, response *FullSpeedPacket,
) bool {
	if !validFullSpeedPacket(request) ||
		response != nil && !validFullSpeedPacket(response) {
		x.words = 0
		return false
	}
	requestStates := request.States
	var responseStates []byte
	if response != nil {
		responseStates = response.States
	}
	needed := 48 + (len(requestStates)-1)*10
	if len(responseStates) != 0 {
		differential := len(responseStates) - 3
		needed += 24 + differential + differential/5
	}
	if needed > len(x.code) {
		x.words = 0
		return false
	}
	index := 0
	emit := func(instruction uint32) {
		x.code[index] = instruction
		index++
	}
	// Dedicated input values are physical full-speed K=1 and J=2.
	emit(0x00100313) // li t1, 1
	emit(0x00200393) // li t2, 2
	// Bound each wait to about four milliseconds so a detached bus cannot
	// strand the caller inside generated code.
	emit(0x00018737)                 // lui a4, 0x18
	emit(rvEncodeI(0, 14, 14, 1696)) // addi a4, a4, 1696 (100000)
	// Wait for idle J, then find the K start edge at three-cycle resolution.
	waitJ := index
	emit(0x804022f3)                  // csrr t0, 0x804
	emit(0x0032f293)                  // andi t0, t0, 3
	emit(rvEncodeBranch(0, 5, 7, 20)) // beq t0, t2, foundJ
	emit(rvEncodeI(0, 14, 14, -1))
	emit(rvEncodeBranch(1, 14, 0, (waitJ-index)*4))
	emit(0x00000513) // li a0, 0
	emit(rvReturn)
	waitK := index
	emit(0x804022f3)
	emit(0x0032f293)
	emit(rvEncodeBranch(0, 5, 6, 20)) // beq t0, t1, foundK
	emit(rvEncodeI(0, 14, 14, -1))
	emit(rvEncodeBranch(1, 14, 0, (waitK-index)*4))
	emit(0x00000513)
	emit(rvReturn)
	// The first sampled K is request[0]. Center the request[1] sample 15
	// cycles later, bounding it to five through seven cycles into that cell.
	emit(0x00000f93) // li t6, 0 (mismatch accumulator)
	for delay := 0; delay < 11; delay++ {
		emit(rvNOP)
	}
	firstSE0 := len(requestStates) - 3
	for symbol := 1; symbol < len(requestStates); symbol++ {
		emit(0x804022f3)
		emit(0x0032f293)
		line := requestStates[symbol]
		if line == 0 {
			emit(rvEncodeR(6, 31, 31, 5)) // or t6, t6, t0
			emit(rvNOP)
		} else {
			expected := uint32(7) // physical J
			if line == 2 {
				expected = 6 // physical K
			}
			emit(rvEncodeR(4, 28, 5, expected)) // xor t3, t0, expected
			emit(rvEncodeR(6, 31, 31, 28))      // or t6, t6, t3
		}
		for delay := 0; delay < 6; delay++ {
			if symbol == firstSE0 && delay == 0 {
				emit(0x7e2025f3) // csrr a1, 0x7e2
			} else {
				emit(rvNOP)
			}
		}
	}
	// Keep the failure return adjacent so the conditional branch is fixed and
	// independent of the size of the generated response.
	if len(responseStates) == 0 {
		emit(rvEncodeBranch(0, 31, 0, 12)) // beq t6, zero, matched
		emit(0x00000513)                   // li a0, 0
		emit(rvReturn)
		emit(0x00100513) // matched: li a0, 1
		emit(rvReturn)
	} else {
		emit(rvEncodeBranch(0, 31, 0, 12)) // beq t6, zero, respond
		emit(0x00000513)                   // li a0, 0
		emit(rvReturn)
		// Reproduce the hardware-calibrated response deadline used by the
		// assembly feasibility probe, then use the calibrated PHY-store cadence.
		emit(0x02e58393) // addi t2, a1, 46
		wait := index
		emit(0x7e202373)                              // csrr t1, 0x7e2
		emit(rvEncodeBranch(6, 6, 7, (wait-index)*4)) // bltu t1, t2, wait
		emit(0x6000f2b7)                              // lui t0, 0x6000f
		emit(0x01c28293)                              // addi t0, t0, 0x1c
		emit(0x00700e93)                              // li t4, 7
		emit(0x00b00f13)                              // li t5, 11
		emit(0x00300f93)                              // li t6, 3
		emit(0x00100313)                              // li t1, 1
		for symbol, line := range responseStates[:len(responseStates)-3] {
			if line == 1 {
				emit(rvStoreT4AtT0)
			} else {
				emit(rvStoreT5AtT0)
			}
			if symbol%5 == 4 {
				emit(rvNOP)
			}
		}
		emit(rvStoreT6AtT0)
		for delay := 0; delay < 10; delay++ {
			emit(rvNOP)
		}
		emit(rvStoreT4AtT0)
		emit(rvStoreT1AtT0)
		emit(0x00100513) // li a0, 1
		emit(rvReturn)
	}
	for clear := index; clear < x.words; clear++ {
		x.code[clear] = 0
	}
	x.words = index
	return true
}

// Entry returns the executable internal-SRAM address of a built exchange.
func (x *FullSpeedExchange) Entry() uintptr {
	if x.words == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&x.code[0]))
}

// Run waits for and matches one host packet, responding inside the generated
// block when configured. The USB PHY must already be owned with dedicated
// inputs enabled and the HP clock fixed at 120 MHz.
func (x *FullSpeedExchange) Run() bool {
	entry := x.Entry()
	if entry == 0 {
		return false
	}
	return RealtimeCall(entry) != 0
}
