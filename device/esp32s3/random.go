package esp32s3

import "renvo.dev/device/mmio"

const randomData = uintptr(0x6003507c)

// Random is the ESP32-S3 hardware random source. The bootloader initializes
// the hardware entropy source before entering the application.
type Random struct{}

// Uint32 combines five samples separated by 100 microseconds.
func (*Random) Uint32() uint32 {
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
