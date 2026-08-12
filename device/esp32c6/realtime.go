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
// internal SRAM. Packet preparation happens outside the packet-critical path;
// transmission contains only scheduled register stores and a return.
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
