//go:build renvo && darwin && !ios && arm64

package graphics

// renvo:linkstatic /System/Library/Frameworks/Metal.framework/Metal,MTLCreateSystemDefaultDevice
func metalCreateSystemDefaultDevice() int { return 0 }

// Aggregate arguments larger than sixteen bytes are passed indirectly by
// AAPCS64. The pointer-shaped declaration matches MTLRegion's native ABI.
// renvo:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func metalGetBytesRaw(object, selector int, bytes *byte, bytesPerRow int, region *metalRegion, level int) int {
	return 0
}

func metalAttachLayer(window *Window, layer int) bool {
	if window == nil || window.view == 0 || layer == 0 {
		return false
	}
	objcMsg1(window.view, selector("setWantsLayer:"), 1)
	objcMsg1(window.view, selector("setLayer:"), layer)
	return true
}

func metalResizeLayer(window *Window, layer, width, height int) {
	objcMsgSize(layer, selector("setDrawableSize:"), width, height)
}
