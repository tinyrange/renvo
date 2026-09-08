package esp32s3

import "renvo.dev/device/mmio"

type clockError string

func (e clockError) Error() string { return string(e) }

// Use160MHzClock promotes an existing 80 MHz PLL boot clock to 160 MHz.
// APB remains 80 MHz, and the 16 MHz system timer is unchanged. Call before
// initializing peripherals. Already faster PLL configurations are preserved.
// This uses the ESP-IDF v5.4.2 rtc_clk_cpu_freq_to_pll_mhz sequence for 160 MHz:
// enable the fifth LDO slave before increasing CPU frequency. Both 80 and
// 160 MHz use the same calibrated bias; 240 MHz needs a separate bias setup.
// https://github.com/espressif/esp-idf/blob/v5.4.2/components/esp_hw_support/port/esp32s3/rtc_clk.c
func Use160MHzClock() error {
	const sysclk = uintptr(0x600c0060)
	const cpu = uintptr(0x600c0010)
	const ldo = uintptr(0x600081fc)
	config := mmio.Load32(cpu)
	if mmio.Load32(sysclk)>>10&3 != 1 || config&3 == 3 {
		return clockError("unsupported CPU clock source")
	}
	if config&3 != 0 {
		return nil
	}
	// DEFAULT_LDO_SLAVE (0x7) >> (160/80) = 1 powered-down slave.
	mmio.Store32(ldo, mmio.Load32(ldo)&^(uint32(0x3f)<<13)|1<<13)
	mmio.Store32(cpu, config&^3|1)
	return nil
}
