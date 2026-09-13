package esp32p4

import "renvo.dev/device/mmio"

// BaseMAC reads the factory IEEE MAC from the eFuse read registers. It does
// not write eFuses. Suitable for the sole Ethernet interface on Unit PoE-P4.
func BaseMAC() [6]byte {
	low := mmio.Load32(0x5012d044)
	high := mmio.Load32(0x5012d048)
	return [6]byte{byte(high >> 8), byte(high), byte(low >> 24), byte(low >> 16), byte(low >> 8), byte(low)}
}
