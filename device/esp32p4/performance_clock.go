package esp32p4

import "renvo.dev/device/mmio"

type clockError string

func (e clockError) Error() string { return string(e) }

// UseFullPLLClock promotes the ESP-IDF bootloader's CPLL/4 setting to CPLL/1
// without recalibrating the PLL. On the Tab5 bootloader this is 90 -> 360 MHz.
// MEM runs at CPU/2 and APB at MEM/2: <=200/100 MHz for a 360/400 MHz CPLL.
// Call before initializing timing-sensitive application peripherals. It is
// deliberately not a general overclocking/frequency-scaling API.
// Sequence: ESP-IDF v5.4.2 rtc_clk_cpu_freq_to_cpll_mhz, upscale APB -> MEM -> CPU.
func UseFullPLLClock() error {
	const base = uintptr(0x500e6000)
	if mmio.Load32(0x50111040)&3 != 1 {
		return clockError("CPU is not using CPLL")
	}
	cpu := mmio.Load32(base + 4)
	mem := mmio.Load32(base + 8)
	apb := mmio.Load32(base + 12)
	if cpu&0x1fffffe0 == 0 && mem == 1 && apb == 1<<16 && mmio.Load32(base+16) == 0 {
		return nil
	}
	// Only accept the integer divide-by-four boot layout. Unknown fractional
	// dividers or bus configurations require an explicit clock-tree transition.
	if cpu&0x1fffffe0 != 3<<5 || mem != 0 || apb != 0 || mmio.Load32(base+16) != 0 {
		return clockError("unsupported boot clock dividers")
	}
	mmio.Store32(base+12, 1<<16)
	updateBusClock()
	mmio.Store32(base+8, 1)
	updateBusClock()
	mmio.Store32(base+4, mmio.Load32(base+4)&^uint32(0x1fffffe0))
	updateBusClock()
	return nil
}

func updateBusClock() {
	const control = uintptr(0x500e6004)
	mmio.Store32(control, mmio.Load32(control)|16)
	for mmio.Load32(control)&16 != 0 {
	}
}
