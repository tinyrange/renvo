package rp2350

import "renvo.dev/device/mmio"

const (
	resetsBase     = uintptr(0x40020000)
	ioBank0Reset   = uint32(1 << 6)
	padsBank0Reset = uint32(1 << 9)
	timer0Reset    = uint32(1 << 23)
)

func releaseReset(mask uint32) {
	mmio.Store32(resetsBase+0x3000, mask)
	for mmio.Load32(resetsBase+8)&mask != mask {
	}
}
