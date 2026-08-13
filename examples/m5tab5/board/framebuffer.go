package board

import (
	"renvo.dev/std/graphics"
	"unsafe"
)

const (
	bridgeBase      = dsiBase + 0x800
	dmaBase         = uintptr(0x50081000)
	dmaChannel      = dmaBase + 0x100
	bridgeFIFO      = uintptr(0x50105000)
	framebufferA    = uintptr(0x48000000)
	framebufferB    = uintptr(0x48600000)
	framePixelCount = DisplayWidth * DisplayHeight
	framebufferSize = framePixelCount * 2
	landscapeWidth  = DisplayHeight
	landscapeHeight = DisplayWidth
	surfaceBase     = uintptr(0x48200000)
	surfaceSize     = landscapeWidth * landscapeHeight * 4
)

var frameWords [2]uint32
var frameSource uintptr
var frameSourceFixed bool
var frontFramebuffer = framebufferA
var backFramebuffer = framebufferB

func enableDMAClock() {
	update32(clockBase+0x14, 0, 1<<13)
	update32(clockBase+0x18, 0, 1<<5)
}

func initializeDMA() {
	enableDMAClock()
	store32(dmaBase+0x58, 1)
	for load32(dmaBase+0x58)&1 != 0 {
	}
	store32(dmaBase+0x10, 3)
}

func configureBridge() {
	// These fields are the ESP-IDF DPI setup expressed directly: 720x1280
	// RGB565, DMA flow control, one block per frame, and a 256-word burst.
	store32(bridgeBase+0x04, 1)
	store32(bridgeBase+0x08, 0x100)
	store32(bridgeBase+0x0c, framePixelCount/4)
	store32(bridgeBase+0x18, 2)
	store32(bridgeBase+0x30, 0x050005f4)
	store32(bridgeBase+0x34, 0x00140018)
	store32(bridgeBase+0x38, 0x02d00322)
	store32(bridgeBase+0x3c, 0x00020028)
	store32(bridgeBase+0x40, 0x00002d00)
	store32(bridgeBase+0x88, 0x10)
	store32(bridgeBase+0x8c, 0x300)
}

func startFrameDMA() {
	store32(dmaChannel+0x00, uint32(frameSource))
	store32(dmaChannel+0x08, uint32(bridgeFIFO))
	store32(dmaChannel+0x10, framePixelCount/4-1)
	// 64-bit source and destination. The framebuffer increments while the
	// bridge FIFO remains fixed. A fixed source is retained for solid probes.
	control := uint32(1 | 1<<6 | 3<<8 | 3<<11 | 8<<14 | 7<<18)
	if frameSourceFixed {
		control |= 1 << 4
	}
	store32(dmaChannel+0x18, control)
	store32(dmaChannel+0x1c, 1<<6|16<<7|1<<15|16<<16)
	// Auto-reload both addresses after every frame. IOC_BLKTFR remains clear,
	// so VDMA does not suspend between blocks waiting for an ISR.
	store32(dmaChannel+0x20, 1|1<<2)
	store32(dmaChannel+0x24, 1|1<<17|4<<23|1<<27)
	store32(dmaChannel+0x98, 0xffffffff)
	store32(dmaChannel+0x9c, 0xffffffff)
	store32(dmaBase+0x18, 0x101)
}

func stopFrameDMA() {
	// Bit eight is the write-enable for channel zero; clearing bit zero stops
	// the reload stream before the front-buffer source is changed.
	store32(dmaBase+0x18, 0x100)
	for load32(dmaBase+0x18)&1 != 0 {
	}
}

// InitSolid replaces the DSI test generator with a bridge/DMA-driven RGB565
// frame which is continuously repeated by a hardware-linked descriptor.
func InitSolid(color uint16) bool {
	if !InitPattern() {
		return false
	}
	word := uint32(color) | uint32(color)<<16
	frameWords[0] = word
	frameWords[1] = word
	frameSource = uintptr(unsafe.Pointer(&frameWords[0]))
	frameSourceFixed = true
	return startScanout()
}

func startScanout() bool {
	initializeDMA()
	configureBridge()
	startFrameDMA()
	update32(dsiBase+0x38, 1<<16, 0)
	store32(dsiBase+0x34, 0)
	store32(bridgeBase+0x40, 0x00002d01)
	store32(bridgeBase+0x44, 1)
	return true
}

func writeBack(address uintptr, size int) {
	const cacheBase = uintptr(0x3ff10000)
	// Push dirty core data lines into L2 first, then dirty L2 lines into PSRAM.
	// P4 ECO2 requires the two cache maps to be synchronized separately.
	for _, cacheMap := range []uint32{16, 32} {
		store32(cacheBase+0x9c, cacheMap)
		store32(cacheBase+0xa0, uint32(address))
		store32(cacheBase+0xa4, uint32(size))
		store32(cacheBase+0x98, 4)
		polls := 0
		for {
			status := load32(cacheBase + 0x98)
			if status&16 != 0 {
				break
			}
			polls++
			if frameSource != 0 && polls&0x3fff == 0 {
				Refresh()
			}
		}
		if frameSource != 0 {
			Refresh()
		}
	}
}

func fillTestFrame(framebuffer uintptr) {
	colors := []uint16{
		0xf800, 0xfd20, 0xffe0, 0x07e0,
		0x07ff, 0x001f, 0x781f, 0xffff,
	}
	for y := 0; y < DisplayHeight; y++ {
		for x := 0; x < DisplayWidth; x++ {
			color := colors[x*len(colors)/DisplayWidth]
			if x < 8 || y < 8 || x >= DisplayWidth-8 || y >= DisplayHeight-8 {
				color = 0xffff
			}
			address := framebuffer + uintptr((y*DisplayWidth+x)*2)
			*(*uint16)(unsafe.Pointer(address)) = color
		}
	}
}

// InitFramebuffer renders a bordered RGB565 color chart into PSRAM, writes
// both cache levels back, and starts incrementing framebuffer scanout.
func InitFramebuffer() bool {
	if !InitPattern() {
		return false
	}
	fillTestFrame(framebufferA)
	writeBack(framebufferA, framebufferSize)
	frontFramebuffer = framebufferA
	backFramebuffer = framebufferB
	frameSource = frontFramebuffer
	frameSourceFixed = false
	return startScanout()
}

func externalBytes(address uintptr, size int) []byte {
	var result []byte
	descriptor := (*[3]uint32)(unsafe.Pointer(&result))
	descriptor[0] = uint32(address)
	descriptor[1] = uint32(size)
	descriptor[2] = uint32(size)
	return result
}

// NewLandscapeSurface returns a 1280 by 720 RGBA surface backed directly by
// Tab5 PSRAM, outside Renvo's small object arena.
func NewLandscapeSurface() *graphics.Surface {
	return graphics.NewSurfaceBuffer(
		landscapeWidth, landscapeHeight, externalBytes(surfaceBase, surfaceSize),
	)
}

// NewPortraitSurface returns a native 720 by 1280 RGBA surface backed directly
// by Tab5 PSRAM.
func NewPortraitSurface() *graphics.Surface {
	return graphics.NewSurfaceBuffer(
		DisplayWidth, DisplayHeight, externalBytes(surfaceBase, surfaceSize),
	)
}

// PresentPortrait converts only pending native-orientation damage and flushes
// it directly into the continuously scanned framebuffer. Sequential writes are
// cache friendly, and VDMA is never stopped for a presentation.
func PresentPortrait(surface *graphics.Surface) bool {
	if surface == nil || surface.Width != DisplayWidth ||
		surface.Height != DisplayHeight || len(surface.Pixels) < surfaceSize {
		return false
	}
	for region := 0; region < surface.DirtyRectCount(); region++ {
		dirty, ok := surface.DirtyRectAt(region)
		if !ok {
			continue
		}
		minX := int(dirty.MinX)
		minY := int(dirty.MinY)
		maxX := int(dirty.MaxX)
		maxY := int(dirty.MaxY)
		if minX < 0 {
			minX = 0
		}
		if minY < 0 {
			minY = 0
		}
		if maxX > DisplayWidth {
			maxX = DisplayWidth
		}
		if maxY > DisplayHeight {
			maxY = DisplayHeight
		}
		if minX >= maxX || minY >= maxY {
			continue
		}
		for y := minY; y < maxY; y++ {
			for x := minX; x < maxX; x++ {
				source := y*surface.Stride + x*4
				red := uint16(surface.Pixels[source])
				green := uint16(surface.Pixels[source+1])
				blue := uint16(surface.Pixels[source+2])
				color := red&0xf8<<8 | green&0xfc<<3 | blue>>3
				address := frontFramebuffer + uintptr((y*DisplayWidth+x)*2)
				*(*uint16)(unsafe.Pointer(address)) = color
			}
		}
		firstPixel := minY*DisplayWidth + minX
		lastPixel := (maxY-1)*DisplayWidth + maxX
		writeBack(frontFramebuffer+uintptr(firstPixel*2), (lastPixel-firstPixel)*2)
	}
	return true
}

// PresentLandscape rotates an RGBA forms surface through the same 90-degree
// mapping used by the official Tab5 LVGL setup and converts it into the native
// back buffer. The unchanged front buffer is refreshed during conversion, and
// the two buffers swap only after the active frame reaches its boundary.
func PresentLandscape(surface *graphics.Surface) bool {
	if surface == nil || surface.Width != landscapeWidth ||
		surface.Height != landscapeHeight || len(surface.Pixels) < surfaceSize {
		return false
	}
	for y := 0; y < landscapeHeight; y++ {
		for x := 0; x < landscapeWidth; x++ {
			if x&63 == 0 {
				Refresh()
			}
			source := y*surface.Stride + x*4
			red := uint16(surface.Pixels[source])
			green := uint16(surface.Pixels[source+1])
			blue := uint16(surface.Pixels[source+2])
			color := red&0xf8<<8 | green&0xfc<<3 | blue>>3
			native := (DisplayHeight-1-x)*DisplayWidth + y
			address := backFramebuffer + uintptr(native*2)
			*(*uint16)(unsafe.Pointer(address)) = color
		}
	}
	Refresh()
	writeBack(backFramebuffer, framebufferSize)
	stopFrameDMA()
	oldFront := frontFramebuffer
	frontFramebuffer = backFramebuffer
	backFramebuffer = oldFront
	frameSource = frontFramebuffer
	startFrameDMA()
	return true
}

// Refresh recovers a stopped auto-reload stream. During normal scanout VDMA
// remains enabled and repeats the current front buffer without CPU service.
func Refresh() {
	if frameSource == 0 {
		return
	}
	if load32(dmaBase+0x18)&1 == 0 {
		startFrameDMA()
	}
}
