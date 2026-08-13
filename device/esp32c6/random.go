package esp32c6

import "renvo.dev/device/mmio"

const (
	randomClock       = uintptr(0x600b2800)
	randomData        = uintptr(0x600b2808)
	randomClockEnable = uint32(1 << 24)
)

// Random is the ESP32-C6 hardware random source.
type Random struct {
}

// Uint32 combines five hardware samples.
func (r *Random) Uint32() uint32 {
	mmio.Store32(randomClock, mmio.Load32(randomClock)|randomClockEnable)
	result := uint32(0)
	timer := SystemTimer{}
	for sample := 0; sample < 5; sample++ {
		started := timer.Ticks()
		for timer.Ticks()-started < 1600 {
		}
		result ^= mmio.Load32(randomData)
	}
	return result
}
