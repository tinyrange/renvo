package board

import (
	"renvo.dev/std/graphics"
	"unsafe"
)

const (
	bridgeBase       = dsiBase + 0x800
	axiICMBase       = uintptr(0x500a4000)
	dmaBase          = uintptr(0x50081000)
	dmaChannel       = dmaBase + 0x100
	bridgeFIFO       = uintptr(0x50105000)
	framebufferA     = uintptr(0x48000000)
	framebufferB     = uintptr(0x48600000)
	framePixelCount  = DisplayWidth * DisplayHeight
	framebufferSize  = framePixelCount * 2
	landscapeWidth   = DisplayHeight
	landscapeHeight  = DisplayWidth
	surfaceBase      = uintptr(0x48200000)
	surfaceSize      = landscapeWidth * landscapeHeight * 4
	cacheLineSize    = uintptr(64)
	displayMailbox   = uintptr(0x4ff40080)
	frameDescriptorA = uintptr(0x4ff40100)
	frameDescriptorB = uintptr(0x4ff40140)
	interruptMatrix  = uintptr(0x500d6000)
	clicBase         = uintptr(0x20800000)
	clicControl      = uintptr(0x20801000)
	dwGDMAInterrupt  = uintptr(24)
	displayCPULine   = uintptr(4)
	displayCLICID    = displayCPULine + 16
	internalNoCache  = uintptr(0x40000000)
	systemTimerBase  = uintptr(0x500e2000)
)

var frameWords [2]uint32
var frameSource uintptr
var frameSourceFixed bool
var frontFramebuffer = framebufferA
var backFramebuffer = framebufferB
var scanoutStarted bool
var scanoutUnderruns uint32

// DisplayStats exposes counters useful when tuning PSRAM bandwidth on the
// physical board. Values are cumulative since framebuffer initialization.
type DisplayStats struct {
	DMA2DCopies      uint32
	ScanoutUnderruns uint32
}

// FramebufferStats returns the current rendering and scanout diagnostics.
func FramebufferStats() DisplayStats {
	Refresh()
	return DisplayStats{
		DMA2DCopies:      dma2DCopies,
		ScanoutUnderruns: scanoutUnderruns,
	}
}

// FrameNumber returns the number of complete hardware scanout frames.
func FrameNumber() uint32 {
	Refresh()
	return load32(displayMailbox + 4)
}

// Milliseconds reads the ESP32-P4's 16 MHz system timer. Its uint32 result
// wraps naturally after roughly 268 seconds and is intended for short deltas.
func Milliseconds() uint32 {
	store32(systemTimerBase+0x04, 1<<30)
	for load32(systemTimerBase+0x04)&(1<<29) == 0 {
	}
	return load32(systemTimerBase+0x44) / 16000
}

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

func configureScanoutQoS() {
	// The arbitration registers are behind the ICM's local register clock,
	// which resets disabled even though traffic already passes through the ICM.
	store32(axiICMBase+0x04, 1)

	// The framebuffer source is on the DW-GDMA memory master while software
	// rendering reaches PSRAM through the cache master. Give both DW-GDMA ports
	// the highest AXI arbitration class so a full-screen paint cannot starve the
	// continuously consumed DSI FIFO. Keep cache and DMA2D only one arbitration
	// class below scanout: a large gap can starve the renderer while the display
	// is active even though it protects the bridge FIFO.
	const qosMask = uint32(0x000ffff0)
	const qosValue = uint32(14<<4 | 14<<8 | 15<<12 | 15<<16)
	update32(axiICMBase+0x28, qosMask, qosValue) // ARQOS
	update32(axiICMBase+0x30, qosMask, qosValue) // AWQOS

}

func configureBridge() {
	// These fields are the ESP-IDF DPI setup expressed directly: 720x1280
	// RGB565, DMA flow control, one block per frame, and a 256-word burst.
	store32(bridgeBase+0x04, 1)
	store32(bridgeBase+0x08, 0x100)
	store32(bridgeBase+0x0c, framePixelCount/4)
	store32(bridgeBase+0x18, 2)
	// Active height 1280, total height 1544 (20 sync + 24 back porch +
	// 1280 active + 220 front porch).
	store32(bridgeBase+0x30, 0x05000608)
	store32(bridgeBase+0x34, 0x00140018)
	// Keep the vendor timing in pixel units. ESP-IDF accepts the integer
	// divider's actual 80 MHz result and retains the requested 802-pixel line.
	store32(bridgeBase+0x38, 0x02d00322)
	store32(bridgeBase+0x3c, 0x00020028)
	store32(bridgeBase+0x40, 0x00002d00)
	store32(bridgeBase+0x88, 0x10)
	store32(bridgeBase+0x8c, 0x300)
}

func frameDescriptor(framebuffer uintptr) uintptr {
	if framebuffer == framebufferB {
		return frameDescriptorB
	}
	return frameDescriptorA
}

func prepareFrameDescriptor(descriptor, framebuffer uintptr) {
	descriptor += internalNoCache
	for offset := uintptr(0); offset < 64; offset += 4 {
		store32(descriptor+offset, 0)
	}
	store32(descriptor, uint32(framebuffer))
	store32(descriptor+8, uint32(bridgeFIFO))
	store32(descriptor+16, framePixelCount/4-1)
	control := uint32(1 | 1<<6 | 3<<8 | 3<<11 | 8<<14 | 7<<18)
	store32(descriptor+32, control)
	// One descriptor is one complete frame. The transfer-done handler restores
	// LAST and VALID before attaching the descriptor for the following frame.
	store32(descriptor+36, 1<<6|16<<7|1<<15|16<<16|1<<30|1<<31)
}

func initializeFrameDescriptors() {
	prepareFrameDescriptor(frameDescriptor(framebufferA), framebufferA)
	prepareFrameDescriptor(frameDescriptor(framebufferB), framebufferB)
}

func initializeDMAInterrupt() {
	// Route the DW-GDMA aggregate source to external CPU interrupt four. CLIC
	// external IDs begin at 16; priority one is encoded in the top byte when
	// nlbits is three. The target startup has already installed the aligned
	// non-vectored handler and enabled mstatus.MIE.
	update32(clicBase, 0x1e, 3<<1)
	store32(clicBase+0x08, 0)
	for interruptID := uintptr(0); interruptID < 48; interruptID++ {
		update32(clicControl+interruptID*4, 1<<8, 0)
	}
	update32(interruptMatrix+dwGDMAInterrupt*4, 0x3f, uint32(displayCLICID))
	store32(clicControl+displayCLICID*4, 1<<29|1<<8)
}

func markFrameDescriptorValid(descriptor uintptr) {
	descriptor += internalNoCache
	store32(descriptor+36, load32(descriptor+36)|1<<30|1<<31)
}

func armFrameDescriptor(descriptor uintptr) {
	markFrameDescriptorValid(descriptor)
	store32(dmaChannel+0x98, 0xffffffff)
	store32(dmaChannel+0x9c, 0xffffffff)
	store32(dmaChannel+0x28, uint32(descriptor)|1)
	store32(dmaBase+0x18, 0x101)
}

func startFrameDMA() {
	if !frameSourceFixed {
		descriptor := frameDescriptor(frameSource)
		store32(displayMailbox, uint32(descriptor))
		store32(displayMailbox+4, 0)
		store32(displayMailbox+8, uint32(descriptor))
		control := uint32(1 | 1<<6 | 3<<8 | 3<<11 | 8<<14 | 7<<18)
		store32(dmaChannel+0x18, control)
		store32(dmaChannel+0x1c, 1<<6|16<<7|1<<15|16<<16)
		store32(dmaChannel+0x20, 3|3<<2)
		store32(dmaChannel+0x24, 1|1<<17|4<<23|1<<27)
		store32(dmaChannel+0x80, 1<<1)
		store32(dmaChannel+0x90, 1<<1)
		armFrameDescriptor(descriptor)
		return
	}
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
	store32(dmaChannel+0x20, 1|1<<2)
	store32(dmaChannel+0x24, 1|1<<17|4<<23|1<<27)
	store32(dmaChannel+0x98, 0xffffffff)
	store32(dmaChannel+0x9c, 0xffffffff)
	store32(dmaBase+0x18, 0x101)
}

func swapFrameDMA(nextFramebuffer uintptr) bool {
	// Publishing only changes the ISR mailbox. The active descriptor is never
	// touched. If this store narrowly misses the current frame boundary, the ISR
	// repeats the old front once and consumes it at the following boundary.
	descriptor := frameDescriptor(nextFramebuffer)
	store32(displayMailbox, uint32(descriptor))
	for load32(displayMailbox+8) != uint32(descriptor) {
		if load32(bridgeBase+0x58)&1 != 0 {
			scanoutUnderruns++
			store32(bridgeBase+0x54, 1)
			return false
		}
	}
	frameSource = nextFramebuffer
	return true
}

func stopFrameDMA() {
	// Bit eight is the write-enable for channel zero. DW-GDMA finishes the
	// current block before acknowledging the channel disable, so the source can
	// be changed only after this loop observes the completed full-frame block.
	store32(dmaBase+0x18, 0x100)
	for load32(dmaBase+0x18)&1 != 0 {
	}
}

// InitSolid replaces the DSI test generator with a bridge/DMA-driven RGB565
// frame which is continuously repeated by a hardware-linked descriptor.
func InitSolid(color uint16) bool {
	if !initDisplay(false) {
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
	configureScanoutQoS()
	configureBridge()
	if !frameSourceFixed {
		initializeFrameDescriptors()
		initializeDMAInterrupt()
	}
	startFrameDMA()
	update32(dsiBase+0x38, 1<<16, 0)
	store32(dsiBase+0x34, 0)
	store32(bridgeBase+0x40, 0x00002d01)
	store32(bridgeBase+0x44, 1)
	scanoutStarted = true
	// Keep the panel dark until its first real DMA-backed frame is active. This
	// prevents the DSI diagnostic generator from flashing during application
	// startup.
	enableBacklight()
	return true
}

func cacheSync(address uintptr, size int, operation uint32) {
	const cacheBase = uintptr(0x3ff10000)
	if size <= 0 {
		return
	}
	end := address + uintptr(size)
	address = address & 0xffffffc0
	end = (end + cacheLineSize - 1) & 0xffffffc0
	size = int(end - address)
	// Push dirty core data lines into L2 first, then dirty L2 lines into PSRAM.
	// P4 ECO2 requires the two cache maps to be synchronized separately.
	for _, cacheMap := range []uint32{16, 32} {
		store32(cacheBase+0x9c, cacheMap)
		store32(cacheBase+0xa0, uint32(address))
		store32(cacheBase+0xa4, uint32(size))
		store32(cacheBase+0x98, operation)
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

func writeBack(address uintptr, size int) {
	cacheSync(address, size, 4)
}

func invalidate(address uintptr, size int) {
	cacheSync(address, size, 1)
}

func writeBackInvalidate(address uintptr, size int) {
	cacheSync(address, size, 8)
}

// InitFramebuffer initializes the panel with its backlight off. Continuous
// framebuffer scanout and the backlight start together on first presentation.
func InitFramebuffer() bool {
	if !initDisplay(false) {
		return false
	}
	frontFramebuffer = framebufferA
	backFramebuffer = framebufferB
	frameSource = 0
	frameSourceFixed = false
	scanoutStarted = false
	return true
}

func bindExternalBytes(result *[]byte, address uintptr, size int) {
	if result == nil {
		return
	}
	// Renvo normalizes each slice header component to an eight-byte backend
	// slot, including on 32-bit targets. Use explicit low/high words so the
	// length cannot become the high half of the data pointer. Initialize the
	// caller's header in place: returning a locally fabricated slice would
	// correctly persist (copy) its backing storage into Renvo's arena.
	descriptor := (*[6]uint32)(unsafe.Pointer(result))
	descriptor[0] = uint32(address)
	descriptor[1] = 0
	descriptor[2] = uint32(size)
	descriptor[3] = 0
	descriptor[4] = uint32(size)
	descriptor[5] = 0
}

// NewLandscapeSurface returns a 1280 by 720 RGBA surface backed directly by
// Tab5 PSRAM, outside Renvo's small object arena.
func NewLandscapeSurface() *graphics.Surface {
	var pixels []byte
	bindExternalBytes(&pixels, surfaceBase, surfaceSize)
	return graphics.NewSurfaceBuffer(
		landscapeWidth, landscapeHeight, pixels,
	)
}

// NewPortraitSurface returns a native 720 by 1280 RGB565 surface backed by the
// Tab5 PSRAM buffer which is not currently being scanned out.
func NewPortraitSurface() *graphics.Surface {
	var pixels []byte
	bindExternalBytes(&pixels, backFramebuffer, framebufferSize)
	return graphics.NewSurfaceBufferFormatPreserve(
		DisplayWidth, DisplayHeight, graphics.PixelRGB565,
		pixels,
	)
}

type rowBand struct {
	minY int
	maxY int
}

func clippedDamage(surface *graphics.Surface, region int) (int, int, int, int, bool) {
	dirty, ok := surface.DirtyRectAt(region)
	if !ok {
		return 0, 0, 0, 0, false
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
	return minX, minY, maxX, maxY, minX < maxX && minY < maxY
}

func damageBands(surface *graphics.Surface) ([64]rowBand, int) {
	var bands [64]rowBand
	count := 0
	for region := 0; region < surface.DirtyRectCount() && count < len(bands); region++ {
		_, minY, _, maxY, ok := clippedDamage(surface, region)
		if !ok {
			continue
		}
		at := count
		for at > 0 && bands[at-1].minY > minY {
			bands[at] = bands[at-1]
			at--
		}
		bands[at] = rowBand{minY: minY, maxY: maxY}
		count++
	}
	merged := 0
	for index := 0; index < count; index++ {
		band := bands[index]
		if merged > 0 && band.minY <= bands[merged-1].maxY+1 {
			if band.maxY > bands[merged-1].maxY {
				bands[merged-1].maxY = band.maxY
			}
			continue
		}
		bands[merged] = band
		merged++
	}
	return bands, merged
}

func writeBackDamage(address uintptr, surface *graphics.Surface) {
	bands, count := damageBands(surface)
	for index := 0; index < count; index++ {
		start := bands[index].minY * surface.Stride
		size := (bands[index].maxY - bands[index].minY) * surface.Stride
		writeBack(address+uintptr(start), size)
	}
}

func copySurfaceDamageDMA2D(surface *graphics.Surface, source, destination []byte) bool {
	if len(source) == 0 || len(destination) == 0 {
		return false
	}
	sourceBase := uintptr(unsafe.Pointer(&source[0]))
	destinationBase := uintptr(unsafe.Pointer(&destination[0]))
	for region := 0; region < surface.DirtyRectCount(); region++ {
		minX, minY, maxX, maxY, ok := clippedDamage(surface, region)
		if !ok {
			continue
		}
		// The old front buffer has no dirty CPU cache ownership. Discard its
		// clean cache rows before and after DMA so the next CPU repaint observes
		// the new generation rather than stale cached pixels.
		start := minY * surface.Stride
		size := (maxY - minY) * surface.Stride
		writeBackInvalidate(destinationBase+uintptr(start), size)
		if !copyRectDMA2D(sourceBase, destinationBase, minX, minY, maxX, maxY) {
			return false
		}
		invalidate(destinationBase+uintptr(start), size)
	}
	return true
}

func copySurfaceRegionDMA2D(
	surface *graphics.Surface, sourceBase, destinationBase uintptr,
	minX, minY, maxX, maxY int,
) bool {
	if minX >= maxX || minY >= maxY {
		return true
	}
	start := minY * surface.Stride
	size := (maxY - minY) * surface.Stride
	writeBackInvalidate(destinationBase+uintptr(start), size)
	if !copyRectDMA2D(sourceBase, destinationBase, minX, minY, maxX, maxY) {
		return false
	}
	invalidate(destinationBase+uintptr(start), size)
	return true
}

func copySurfaceDamageDMA2DOutside(
	surface *graphics.Surface, source, destination []byte, excludedTop, excludedBottom int,
) bool {
	if len(source) == 0 || len(destination) == 0 {
		return false
	}
	sourceBase := uintptr(unsafe.Pointer(&source[0]))
	destinationBase := uintptr(unsafe.Pointer(&destination[0]))
	for region := 0; region < surface.DirtyRectCount(); region++ {
		minX, minY, maxX, maxY, ok := clippedDamage(surface, region)
		if !ok {
			continue
		}
		if minY < excludedTop {
			topEnd := maxY
			if topEnd > excludedTop {
				topEnd = excludedTop
			}
			if !copySurfaceRegionDMA2D(surface, sourceBase, destinationBase, minX, minY, maxX, topEnd) {
				return false
			}
		}
		if maxY > excludedBottom {
			bottomStart := minY
			if bottomStart < excludedBottom {
				bottomStart = excludedBottom
			}
			if !copySurfaceRegionDMA2D(surface, sourceBase, destinationBase, minX, bottomStart, maxX, maxY) {
				return false
			}
		}
	}
	return true
}

func copyWholeFramebuffer(source, destination uintptr) bool {
	// Establish an identical generation in both buffers before incremental
	// damage rendering begins. DMA2D is required for this display pipeline.
	writeBackInvalidate(destination, framebufferSize)
	if copyRectDMA2D(source, destination, 0, 0, DisplayWidth, DisplayHeight) {
		invalidate(destination, framebufferSize)
		return true
	}
	return false
}

func presentPortrait(surface *graphics.Surface, synchronizeDamage bool, excludedTop, excludedBottom int) bool {
	if surface == nil || surface.Width != DisplayWidth ||
		surface.Height != DisplayHeight || surface.Format != graphics.PixelRGB565 ||
		surface.Stride != DisplayWidth*2 || len(surface.Pixels) < framebufferSize {
		return false
	}
	if surface.DirtyRectCount() == 0 {
		return true
	}
	base := uintptr(unsafe.Pointer(&surface.Pixels[0]))
	if base != backFramebuffer {
		return false
	}
	writeBackDamage(backFramebuffer, surface)
	if !scanoutStarted {
		firstFront := backFramebuffer
		firstBack := frontFramebuffer
		if !copyWholeFramebuffer(firstFront, firstBack) {
			return false
		}
		frontFramebuffer = firstFront
		backFramebuffer = firstBack
		frameSource = frontFramebuffer
		if !startScanout() {
			return false
		}
		bindExternalBytes(&surface.Pixels, backFramebuffer, framebufferSize)
		return true
	}
	oldFront := frontFramebuffer
	if !swapFrameDMA(backFramebuffer) {
		return false
	}
	frontFramebuffer = backFramebuffer
	backFramebuffer = oldFront
	frameSource = frontFramebuffer

	if synchronizeDamage {
		var source []byte
		var destination []byte
		bindExternalBytes(&source, frontFramebuffer, framebufferSize)
		bindExternalBytes(&destination, backFramebuffer, framebufferSize)
		if excludedTop < excludedBottom {
			if !copySurfaceDamageDMA2DOutside(surface, source, destination, excludedTop, excludedBottom) {
				return false
			}
		} else {
			if !copySurfaceDamageDMA2D(surface, source, destination) {
				return false
			}
		}
	}
	bindExternalBytes(&surface.Pixels, backFramebuffer, framebufferSize)
	return true
}

// PresentPortrait publishes a native RGB565 back buffer at a full-frame DMA
// boundary. After the flip, changed regions are copied into the old front so
// an ordinary incremental renderer starts from the latest generation.
func PresentPortrait(surface *graphics.Surface) bool {
	return presentPortrait(surface, true, 0, 0)
}

// PresentPortraitRetained publishes without copying damage into the old front.
// A retained renderer should repaint the union of the current frame's damage
// and the preceding frame's damage before each call. This avoids a synchronous
// post-flip DMA transaction while keeping both buffers generation-correct.
func PresentPortraitRetained(surface *graphics.Surface) bool {
	return presentPortrait(surface, false, 0, 0)
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
	writeBack(backFramebuffer, framebufferSize)
	if !scanoutStarted {
		firstFront := backFramebuffer
		firstBack := frontFramebuffer
		copyWholeFramebuffer(firstFront, firstBack)
		frontFramebuffer = firstFront
		backFramebuffer = firstBack
		frameSource = frontFramebuffer
		return startScanout()
	}
	oldFront := frontFramebuffer
	if !swapFrameDMA(backFramebuffer) {
		return false
	}
	frontFramebuffer = backFramebuffer
	backFramebuffer = oldFront
	frameSource = frontFramebuffer
	return true
}

// Refresh records bridge underruns and recovers a channel if an interrupt was
// lost. During normal scanout the transfer-done handler re-arms every frame.
func Refresh() {
	if load32(bridgeBase+0x58)&1 != 0 {
		scanoutUnderruns++
		store32(bridgeBase+0x54, 1)
	}
	if frameSource == 0 || !scanoutStarted {
		return
	}
	// Descriptor-backed scanout is owned exclusively by the transfer-done ISR.
	// Channel enable is briefly clear at every frame boundary; treating that
	// normal window as a lost interrupt races the ISR and can attach a link while
	// it is already being re-armed. Fixed-source probes do not use that ISR.
	if frameSourceFixed && load32(dmaBase+0x18)&1 == 0 {
		startFrameDMA()
	}
}
