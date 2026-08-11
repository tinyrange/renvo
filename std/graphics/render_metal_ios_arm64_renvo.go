//go:build renvo && ios && arm64

package graphics

// renvo:linkstatic /System/Library/Frameworks/Metal.framework/Metal,_MTLCreateSystemDefaultDevice
func metalCreateSystemDefaultDevice() int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_objc_msgSend
func metalGetBytesRaw(object, selector int, bytes *byte, bytesPerRow int, region *metalRegion, level int) int {
	return 0
}

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_objc_msgSend
func objcMsg3(object, selector, a, b, c int) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_objc_msgSend
func objcMsgPointer3(object, selector int, value *byte, length, options int) int { return 0 }

// CGSize arguments occupy floating-point registers in the Objective-C ABI.
// renvo:linkstatic renvo-ios,renvoIOSObjcMsgSize
func objcMsgSize(object, selector, width, height int) int { return 0 }

func objcGetClass(name string) int { return iosClass(name) }
func selector(name string) int     { return iosSelector(name) }
func cocoaString(text string) int  { return iosString(text) }

func objcMsg0(object, selector int) int       { return iosObjcMsg0(object, selector) }
func objcMsg1(object, selector, a int) int    { return iosObjcMsg1(object, selector, a) }
func objcMsg2(object, selector, a, b int) int { return iosObjcMsg2(object, selector, a, b) }

func darwinNow() Scalar { return 0 }

func metalAttachLayer(window *Window, layer int) bool {
	if window == nil || window.view == 0 || layer == 0 {
		return false
	}
	root := iosObjcMsg0(window.view, iosSelector("layer"))
	if root == 0 {
		return false
	}
	iosObjcMsg1(layer, iosSelector("setOpaque:"), 1)
	iosObjcMsg1(root, iosSelector("addSublayer:"), layer)
	return true
}

func metalResizeLayer(window *Window, layer, width, height int) {
	objcMsgSize(layer, iosSelector("setDrawableSize:"), width, height)
	if window != nil {
		frameWidth := iosObjcMsgRectWidth(window.view, iosSelector("bounds"))
		frameHeight := iosObjcMsgRectHeight(window.view, iosSelector("bounds"))
		if frameWidth <= 0 {
			frameWidth = window.width
		}
		if frameHeight <= 0 {
			frameHeight = window.height
		}
		iosObjcMsgRect(layer, iosSelector("setFrame:"), 0, 0, frameWidth, frameHeight)
	}
}
