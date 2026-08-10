package board

const (
	randomClock       = uintptr(0x600b2800)
	randomData        = uintptr(0x600b2808)
	randomClockEnable = uint32(1 << 24)
)

// Random32 samples the ESP32-C6 hardware RNG. The bootloader seeds its internal
// state from physical entropy; spacing and combining several reads avoids
// draining that state faster than the hardware can update it.
func Random32() uint32 {
	store32(randomClock, load32(randomClock)|randomClockEnable)
	result := uint32(0)
	for sample := 0; sample < 5; sample++ {
		DelayMicroseconds(100)
		result = result ^ load32(randomData)
	}
	return result
}
