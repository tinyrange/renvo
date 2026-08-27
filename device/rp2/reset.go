package rp2

import "renvo.dev/device/mmio"

const (
	rp2040Resets = uintptr(0x4000c000)
	rp2350Resets = uintptr(0x40020000)

	rp2040IOBank0Reset   = uint32(1 << 5)
	rp2040PadsBank0Reset = uint32(1 << 8)
	rp2040PIO0Reset      = uint32(1 << 10)
	rp2040TimerReset     = uint32(1 << 21)

	rp2350IOBank0Reset   = uint32(1 << 6)
	rp2350PadsBank0Reset = uint32(1 << 9)
	rp2350PIO0Reset      = uint32(1 << 11)
	rp2350Timer0Reset    = uint32(1 << 23)
)

func releaseReset(rp2040Mask, rp2350Mask uint32) {
	base, mask := rp2040Resets, rp2040Mask
	if isRP2350() {
		base, mask = rp2350Resets, rp2350Mask
	}
	mmio.Store32(base+0x3000, mask)
	for mmio.Load32(base+8)&mask != mask {
	}
}
