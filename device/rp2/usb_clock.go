package rp2

import "renvo.dev/device/mmio"

// ConfigureUSBClock establishes the 48 MHz peripheral clock required by the
// USB controller. Literal per-family sequences keep MMIO operands live across
// no helper calls and make the initialization directly auditable.
func ConfigureUSBClock() {
	if isRP2350() {
		mmio.Store32(0x40048000, 0xaa0)
		mmio.Store32(0x4004800c, 47)
		mmio.Store32(0x4004a000, 0xfab000)
		for mmio.Load32(0x40048004)&(1<<31) == 0 {
		}
		mmio.Store32(0x4001303c, 1)
		for mmio.Load32(0x40010044)&1 == 0 {
		}
		mmio.Store32(0x40013030, 3)
		for mmio.Load32(0x40010038)&1 == 0 {
		}
		if mmio.Load32(0x40058000)&(1<<31) == 0 {
			mmio.Store32(0x40022000, 1<<15)
			mmio.Store32(0x40023000, 1<<15)
			for mmio.Load32(0x40020008)&(1<<15) == 0 {
			}
			mmio.Store32(0x40058000, 1)
			mmio.Store32(0x40058008, 100)
			mmio.Store32(0x4005b004, 1<<0|1<<5)
			for mmio.Load32(0x40058000)&(1<<31) == 0 {
			}
			mmio.Store32(0x4005800c, 5<<16|5<<12)
			mmio.Store32(0x4005b004, 1<<3)
		}
		mmio.Store32(0x40010064, 1<<16)
		mmio.Store32(0x40010040, 1<<16)
		mmio.Store32(0x40010060, 1<<11)
		mmio.Store32(0x4001003c, 1<<5|1)
		return
	}

	mmio.Store32(0x40024000, 0xaa0)
	mmio.Store32(0x4002400c, 47)
	mmio.Store32(0x40026000, 0xfab000)
	for mmio.Load32(0x40024004)&(1<<31) == 0 {
	}
	mmio.Store32(0x4000b03c, 1)
	for mmio.Load32(0x40008044)&1 == 0 {
	}
	mmio.Store32(0x4000b030, 3)
	for mmio.Load32(0x40008038)&1 == 0 {
	}
	if mmio.Load32(0x4002c000)&(1<<31) == 0 {
		mmio.Store32(0x4000e000, 1<<13)
		mmio.Store32(0x4000f000, 1<<13)
		for mmio.Load32(0x4000c008)&(1<<13) == 0 {
		}
		mmio.Store32(0x4002c000, 1)
		mmio.Store32(0x4002c008, 40)
		mmio.Store32(0x4002f004, 1<<0|1<<5)
		for mmio.Load32(0x4002c000)&(1<<31) == 0 {
		}
		mmio.Store32(0x4002c00c, 5<<16|2<<12)
		mmio.Store32(0x4002f004, 1<<3)
	}
	mmio.Store32(0x40008058, 1<<8)
	mmio.Store32(0x40008040, 1<<8)
	mmio.Store32(0x40008054, 1<<11)
	mmio.Store32(0x4000803c, 1<<5|1)
}
