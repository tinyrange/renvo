//go:build m5tab5

package board

const (
	dma2DBase         = uintptr(0x50088000)
	dma2DTX           = dma2DBase
	dma2DRX           = dma2DBase + 0x500
	dma2DTXDescriptor = uintptr(0x4ff40180)
	dma2DRXDescriptor = uintptr(0x4ff401c0)
)

var dma2DInitialized bool
var dma2DFailed bool
var dma2DCopies uint32

func initializeDMA2D() {
	if dma2DInitialized {
		return
	}
	// Match dma2d_ll_enable_bus_clock and dma2d_ll_reset_register. The previous
	// sequence cleared the module clock and left the system reset asserted,
	// guaranteeing that every transfer timed out and silently used the CPU.
	update32(clockBase+0x18, 0, 1<<6)
	update32(clockBase+0xc0, 0, 1<<31)
	update32(clockBase+0xc0, 1<<31, 0)
	// Force the register clock on and reset both AXI data FIFOs.
	store32(dma2DBase+0xa04, 4)
	store32(dma2DBase+0xa04, 7)
	store32(dma2DBase+0xa04, 4)
	dma2DInitialized = true
}

func resetDMA2DChannel(channel uintptr, receive bool) bool {
	stopBit := uint32(1 << 20)
	resetAvailable := uint32(1 << 24)
	if receive {
		stopBit = 1 << 21
		resetAvailable = 1 << 23
	}
	store32(channel+0x1c, stopBit)
	// Espressif's LL requires CMD_DISABLE while reset is asserted, followed by
	// clearing both bits. Do not leave the channel disabled after reset.
	update32(channel, 0, 1<<25)
	for attempts := 0; attempts < 1000000; attempts++ {
		if load32(channel+0x24)&resetAvailable != 0 {
			update32(channel, 0, 1<<24)
			update32(channel, 1<<24, 0)
			update32(channel, 1<<25, 0)
			return true
		}
	}
	update32(channel, 1<<25, 0)
	return false
}

func prepareDMA2DDescriptor(
	descriptor uintptr, buffer uintptr,
	pictureWidth, pictureHeight, blockWidth, blockHeight, x, y int,
) {
	// Like DW-GDMA link items, DMA2D descriptors live at normal internal SRAM
	// addresses for the peripheral but are populated through P4's non-cacheable
	// alias so no dirty CPU line can hide or later overwrite their contents.
	descriptor += internalNoCache
	// RGB565 is pbyte=3. Both descriptors are terminal, valid, 2D blocks
	// owned by DMA. The source and destination use identical image geometry.
	word0 := uint32(blockHeight&0x3fff) |
		uint32(blockWidth&0x3fff)<<14 | 1<<29 | 1<<30 | 1<<31
	word1 := uint32(pictureHeight&0x3fff) |
		uint32(pictureWidth&0x3fff)<<14 | 3<<28
	word2 := uint32(y&0x3fff) | uint32(x&0x3fff)<<14
	store32(descriptor, word0)
	store32(descriptor+4, word1)
	store32(descriptor+8, word2)
	store32(descriptor+12, uint32(buffer))
	store32(descriptor+16, 0)
	store32(descriptor+20, 0)
}

func copyRectDMA2DAt(
	source, destination uintptr,
	sourceX, sourceY, destinationX, destinationY, width, height int,
) bool {
	if dma2DFailed {
		return false
	}
	if width <= 0 || height <= 0 {
		return true
	}
	initializeDMA2D()
	if !resetDMA2DChannel(dma2DTX, false) || !resetDMA2DChannel(dma2DRX, true) {
		dma2DFailed = true
		return false
	}

	txDescriptor := dma2DTXDescriptor
	rxDescriptor := dma2DRXDescriptor
	prepareDMA2DDescriptor(txDescriptor, source, DisplayWidth, DisplayHeight, width, height, sourceX, sourceY)
	prepareDMA2DDescriptor(rxDescriptor, destination, DisplayWidth, DisplayHeight, width, height, destinationX, destinationY)
	// M2M uses the first free peripheral selectors from the hardware's
	// documented ranges: TX=4 and RX=3. A 128-byte burst with page-boundary
	// protection is the ESP-IDF default for external-memory copies.
	store32(dma2DTX, 1<<1|4<<6|3<<9|1<<12)
	store32(dma2DRX, 1|4<<6|3<<9|1<<12)
	store32(dma2DTX+0x38, 4)
	store32(dma2DRX+0x38, 3)
	store32(dma2DTX+0x10, 0x1fff)
	store32(dma2DRX+0x10, 0x3fff)
	store32(dma2DTX+0x20, uint32(txDescriptor))
	store32(dma2DRX+0x20, uint32(rxDescriptor))
	store32(dma2DTX+0x1c, 1<<21)
	store32(dma2DRX+0x1c, 1<<22)

	for attempts := 0; attempts < 50000000; attempts++ {
		status := load32(dma2DRX + 4)
		if status&2 != 0 {
			store32(dma2DTX+0x10, 0x1fff)
			store32(dma2DRX+0x10, 0x3fff)
			dma2DCopies++
			return true
		}
		if status&0x3ffc != 0 {
			break
		}
	}
	resetDMA2DChannel(dma2DTX, false)
	resetDMA2DChannel(dma2DRX, true)
	// Latch a hard rendering failure. Tab5 presentation requires DMA2D; it must
	// never disguise a broken accelerator by changing to a PSRAM-heavy CPU path.
	dma2DFailed = true
	return false
}

func copyRectDMA2D(source, destination uintptr, minX, minY, maxX, maxY int) bool {
	return copyRectDMA2DAt(
		source, destination,
		minX, minY, minX, minY, maxX-minX, maxY-minY,
	)
}
