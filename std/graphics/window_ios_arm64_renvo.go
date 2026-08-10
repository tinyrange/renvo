//go:build renvo && ios && arm64

package graphics

const iosUIKit = "/System/Library/Frameworks/UIKit.framework/UIKit"
const iosCoreGraphics = "/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics"
const iosObjC = "/usr/lib/libobjc.A.dylib"
const iosSelfLibrary = "renvo-ios"

const iosBitmapRGBA = 0x4001

var iosWindow *Window

// renvo:linkstatic /System/Library/Frameworks/UIKit.framework/UIKit,_UIApplicationMain
func iosUIApplicationMain(argc, argv, principalClass, delegateClass int) int { return 1 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_objc_getClass
func iosObjcGetClass(name *byte) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_sel_registerName
func iosSelRegisterName(name *byte) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_objc_allocateClassPair
func iosAllocateClassPair(superclass int, name *byte, extraBytes int) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_objc_registerClassPair
func iosRegisterClassPair(class int) {}

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_class_addMethod
func iosClassAddMethod(class, selector, implementation int, types *byte) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_objc_msgSend
func iosObjcMsg0(object, selector int) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_objc_msgSend
func iosObjcMsg1(object, selector, a int) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_objc_msgSend
func iosObjcMsg2(object, selector, a, b int) int { return 0 }

// renvo:linkstatic /usr/lib/libobjc.A.dylib,_objc_msgSend
func iosObjcMsgPointer(object, selector int, a *byte) int { return 0 }

// renvo:linkstatic renvo-ios,renvoIOSObjcMsgRect
func iosObjcMsgRect(object, selector, x, y, width, height int) int { return 0 }

// renvo:linkstatic renvo-ios,renvoIOSObjcMsgRectWidth
func iosObjcMsgRectWidth(object, selector int) int { return 0 }

// renvo:linkstatic renvo-ios,renvoIOSObjcMsgRectHeight
func iosObjcMsgRectHeight(object, selector int) int { return 0 }

// renvo:linkstatic renvo-ios,renvoIOSObjcMsgPointX
func iosObjcMsgPointX(object, selector, view int) int { return 0 }

// renvo:linkstatic renvo-ios,renvoIOSObjcMsgPointY
func iosObjcMsgPointY(object, selector, view int) int { return 0 }

// renvo:linkstatic /System/Library/Frameworks/CoreGraphics.framework/CoreGraphics,_CGDataProviderCreateWithData
func iosDataProviderCreate(info int, data *byte, size, releaseData int) int { return 0 }

// renvo:linkstatic /System/Library/Frameworks/CoreGraphics.framework/CoreGraphics,_CGDataProviderRelease
func iosDataProviderRelease(provider int) {}

// renvo:linkstatic /System/Library/Frameworks/CoreGraphics.framework/CoreGraphics,_CGColorSpaceCreateDeviceRGB
func iosColorSpaceCreateDeviceRGB() int { return 0 }

// renvo:linkstatic /System/Library/Frameworks/CoreGraphics.framework/CoreGraphics,_CGColorSpaceRelease
func iosColorSpaceRelease(space int) {}

// renvo:linkstatic /System/Library/Frameworks/CoreGraphics.framework/CoreGraphics,_CGImageCreate
func iosImageCreate(width, height, bitsPerComponent, bitsPerPixel, bytesPerRow,
	space, bitmapInfo, provider, decode, interpolate, intent int) int {
	return 0
}

// renvo:linkstatic /System/Library/Frameworks/CoreGraphics.framework/CoreGraphics,_CGImageRelease
func iosImageRelease(image int) {}

// renvo:linkstatic renvo-ios,renvoIOSDelegateDidFinishCallback
func iosDelegateDidFinishCallback() int { return 0 }

// renvo:linkstatic renvo-ios,renvoIOSTouchesBeganCallback
func iosTouchesBeganCallback() int { return 0 }

// renvo:linkstatic renvo-ios,renvoIOSTouchesMovedCallback
func iosTouchesMovedCallback() int { return 0 }

// renvo:linkstatic renvo-ios,renvoIOSTouchesEndedCallback
func iosTouchesEndedCallback() int { return 0 }

// renvo:linkstatic renvo-ios,renvoIOSTouchesCancelledCallback
func iosTouchesCancelledCallback() int { return 0 }

func iosCString(text string) []byte {
	value := make([]byte, len(text)+1)
	for i := 0; i < len(text); i++ {
		value[i] = text[i]
	}
	return value
}

func iosClass(name string) int {
	value := iosCString(name)
	return iosObjcGetClass(&value[0])
}

func iosSelector(name string) int {
	value := iosCString(name)
	return iosSelRegisterName(&value[0])
}

func iosString(text string) int {
	value := iosCString(text)
	return iosObjcMsgPointer(iosClass("NSString"),
		iosSelector("stringWithUTF8String:"), &value[0])
}

func iosAddMethod(class int, name string, implementation int, types string) bool {
	encoding := iosCString(types)
	result := iosClassAddMethod(class, iosSelector(name), implementation,
		&encoding[0])
	if result == 0 {
		return false
	}
	return true
}

func iosBuildClass(name, superclass string) int {
	if existing := iosClass(name); existing != 0 {
		return existing
	}
	className := iosCString(name)
	return iosAllocateClassPair(iosClass(superclass), &className[0], 0)
}

func iosRegisterRuntimeClasses() bool {
	delegate := iosBuildClass("RenvoAppDelegate", "NSObject")
	if delegate == 0 || !iosAddMethod(delegate,
		"application:didFinishLaunchingWithOptions:",
		iosDelegateDidFinishCallback(), "c@:@@") {
		return false
	}
	iosRegisterClassPair(delegate)

	view := iosBuildClass("RenvoSurfaceView", "UIImageView")
	if view == 0 ||
		!iosAddMethod(view, "touchesBegan:withEvent:",
			iosTouchesBeganCallback(), "v@:@@") ||
		!iosAddMethod(view, "touchesMoved:withEvent:",
			iosTouchesMovedCallback(), "v@:@@") ||
		!iosAddMethod(view, "touchesEnded:withEvent:",
			iosTouchesEndedCallback(), "v@:@@") ||
		!iosAddMethod(view, "touchesCancelled:withEvent:",
			iosTouchesCancelledCallback(), "v@:@@") {
		return false
	}
	iosRegisterClassPair(view)
	return true
}

func iosRetainCallbacks() {
	// Keep Objective-C entry points reachable from the linked program. UIKit
	// calls the materialized pointers, so ordinary Go call analysis cannot see
	// these edges.
	if iosWindow != nil && iosWindow.native == -1 {
		iosDidFinishLaunching(0, 0, 0, 0)
		iosTouchesBegan(0, 0, 0, 0)
		iosTouchesMoved(0, 0, 0, 0)
		iosTouchesEnded(0, 0, 0, 0)
		iosTouchesCancelled(0, 0, 0, 0)
	}
}

func NewWindow(options WindowOptions) *Window {
	clearLastWindowError()
	if options.Width <= 0 || options.Height <= 0 {
		setLastWindowError("window dimensions must be positive", 0)
		return nil
	}
	if iosWindow != nil && !iosWindow.closed {
		setLastWindowError("iOS supports one application window", 0)
		return nil
	}
	w := &Window{
		width: options.Width, height: options.Height,
		active: true, shown: !options.Hidden,
	}
	w.surface = NewSurface(w.width, w.height)
	iosWindow = w
	iosRetainCallbacks()
	return w
}

// RunIOSApplication transfers the main thread to UIKit after Forms has built
// its retained control tree and installed the window event handler.
func RunIOSApplication(window *Window) int {
	if window == nil || window != iosWindow || !iosRegisterRuntimeClasses() {
		return 1
	}
	pool := iosObjcMsg0(iosClass("NSAutoreleasePool"), iosSelector("new"))
	status := iosUIApplicationMain(0, 0, 0, iosString("RenvoAppDelegate"))
	if pool != 0 {
		iosObjcMsg0(pool, iosSelector("drain"))
	}
	return status
}

// Parameters are intentionally reversed: Renvo's AArch64 callback ABI maps
// the final source parameter to C x0, matching Objective-C's self argument.
func iosDidFinishLaunching(options, application, selector, self int) int {
	w := iosWindow
	if w == nil || w.closed {
		return 0
	}
	screen := iosObjcMsg0(iosClass("UIScreen"), iosSelector("mainScreen"))
	width := iosObjcMsgRectWidth(screen, iosSelector("bounds"))
	height := iosObjcMsgRectHeight(screen, iosSelector("bounds"))
	if width <= 0 {
		width = w.width
	}
	if height <= 0 {
		height = w.height
	}

	native := iosObjcMsg0(iosClass("UIWindow"), iosSelector("alloc"))
	native = iosObjcMsgRect(native, iosSelector("initWithFrame:"),
		0, 0, width, height)
	view := iosObjcMsg0(iosClass("RenvoSurfaceView"), iosSelector("new"))
	controller := iosObjcMsg0(iosClass("UIViewController"), iosSelector("new"))
	if native == 0 || view == 0 || controller == 0 {
		return 0
	}
	iosObjcMsg1(view, iosSelector("setUserInteractionEnabled:"), 1)
	iosObjcMsg1(view, iosSelector("setContentMode:"), 0)
	iosObjcMsg1(controller, iosSelector("setView:"), view)
	iosObjcMsg1(native, iosSelector("setRootViewController:"), controller)
	iosObjcMsg0(native, iosSelector("makeKeyAndVisible"))
	w.app = application
	w.native = native
	w.view = view
	w.context = controller
	w.focused = true
	w.queue(Event{Type: EventWindowFocusGained})
	w.queue(Event{Type: EventWindowExpose,
		Dirty: R(0, 0, Scalar(w.width), Scalar(w.height))})
	iosDispatchQueuedEvents(w)
	return 1
}

func iosQueuePointer(kind EventType, touches int) {
	w := iosWindow
	if w == nil || w.closed || w.view == 0 || touches == 0 {
		return
	}
	touch := iosObjcMsg0(touches, iosSelector("anyObject"))
	if touch == 0 {
		return
	}
	viewWidth := iosObjcMsgRectWidth(w.view, iosSelector("bounds"))
	viewHeight := iosObjcMsgRectHeight(w.view, iosSelector("bounds"))
	if viewWidth <= 0 || viewHeight <= 0 {
		return
	}
	x := iosObjcMsgPointX(touch, iosSelector("locationInView:"), w.view)
	y := iosObjcMsgPointY(touch, iosSelector("locationInView:"), w.view)
	event := Event{Type: kind, X: Scalar(x * w.width / viewWidth),
		Y: Scalar(y * w.height / viewHeight), Button: 1}
	if kind == EventPointerMove && len(w.events) > w.eventHead &&
		w.events[len(w.events)-1].Type == EventPointerMove {
		w.events[len(w.events)-1] = event
	} else {
		w.queue(event)
	}
	iosDispatchQueuedEvents(w)
}

func iosTouchesBegan(event, touches, selector, self int) {
	iosQueuePointer(EventPointerDown, touches)
}

func iosTouchesMoved(event, touches, selector, self int) {
	iosQueuePointer(EventPointerMove, touches)
}

func iosTouchesEnded(event, touches, selector, self int) {
	iosQueuePointer(EventPointerUp, touches)
}

func iosTouchesCancelled(event, touches, selector, self int) {
	iosQueuePointer(EventPointerCancel, touches)
}

func iosDispatchQueuedEvents(w *Window) {
	if w != nil && w.EventHandler != nil {
		w.EventHandler()
	}
}

func (w *Window) Present() bool {
	if w == nil || w.closed || w.surface == nil {
		return false
	}
	if w.view == 0 || !w.surface.dirtyValid {
		return true
	}
	if len(w.surface.Pixels) == 0 {
		return false
	}
	provider := iosDataProviderCreate(0, &w.surface.Pixels[0],
		len(w.surface.Pixels), 0)
	space := iosColorSpaceCreateDeviceRGB()
	if provider == 0 || space == 0 {
		if provider != 0 {
			iosDataProviderRelease(provider)
		}
		if space != 0 {
			iosColorSpaceRelease(space)
		}
		return false
	}
	image := iosImageCreate(w.surface.Width, w.surface.Height, 8, 32,
		w.surface.Stride, space, iosBitmapRGBA, provider, 0, 0, 0)
	if image != 0 {
		uiImage := iosObjcMsg1(iosClass("UIImage"),
			iosSelector("imageWithCGImage:"), image)
		if uiImage != 0 {
			iosObjcMsg1(w.view, iosSelector("setImage:"), uiImage)
		}
		iosImageRelease(image)
	}
	iosColorSpaceRelease(space)
	iosDataProviderRelease(provider)
	if image == 0 {
		return false
	}
	w.surface.ResetDirty()
	return true
}

func (w *Window) Poll() (Event, bool) { return w.nextQueuedEvent() }
func (w *Window) Wait() (Event, bool) { return w.Poll() }

func (w *Window) PollInto(event *Event) bool {
	if w == nil || event == nil {
		return false
	}
	next, ok := w.nextQueuedEvent()
	if !ok {
		return false
	}
	*event = next
	return true
}

func (w *Window) ReadPixels() *Image {
	if w == nil || w.closed || w.surface == nil {
		return nil
	}
	return NewImage(w.surface.Width, w.surface.Height, w.surface.Pixels)
}

func (w *Window) SetTitle(title string) bool { return w != nil && !w.closed }

func (w *Window) Show() bool {
	if w == nil || w.closed {
		return false
	}
	w.shown = true
	if w.native != 0 {
		iosObjcMsg0(w.native, iosSelector("makeKeyAndVisible"))
	}
	return true
}

func (w *Window) Hide() bool {
	if w == nil || w.closed {
		return false
	}
	w.shown = false
	if w.native != 0 {
		iosObjcMsg1(w.native, iosSelector("setHidden:"), 1)
	}
	return true
}

func (w *Window) SetSize(width, height int) bool {
	if w == nil || w.closed || width <= 0 || height <= 0 {
		return false
	}
	w.width, w.height = width, height
	w.surface.Resize(width, height)
	w.queue(Event{Type: EventWindowResize,
		Dirty: R(0, 0, Scalar(width), Scalar(height))})
	w.queue(Event{Type: EventWindowExpose,
		Dirty: R(0, 0, Scalar(width), Scalar(height))})
	iosDispatchQueuedEvents(w)
	return true
}

func (w *Window) RequestRepaint(rect Rect) {
	if w != nil && !w.closed {
		w.queue(Event{Type: EventWindowExpose, Dirty: rect})
		iosDispatchQueuedEvents(w)
	}
}

func (w *Window) SetPointerCapture(captured bool) bool {
	if w == nil || w.closed {
		return false
	}
	w.captured = captured
	return true
}

func (w *Window) SetCursor(cursor Cursor) bool {
	if w == nil || w.closed {
		return false
	}
	w.cursor = cursor
	return true
}

func (w *Window) SetTimer(id int, seconds Scalar) bool { return false }
func (w *Window) CancelTimer(id int)                   {}

func SetClipboardText(text string) bool {
	pasteboard := iosObjcMsg0(iosClass("UIPasteboard"),
		iosSelector("generalPasteboard"))
	if pasteboard == 0 {
		return false
	}
	iosObjcMsg1(pasteboard, iosSelector("setString:"), iosString(text))
	return true
}

func ClipboardText() (string, bool) { return "", false }

func (w *Window) Close() {
	if w == nil || w.closed {
		return
	}
	w.closed = true
	w.active = false
	w.shown = false
	if w.native != 0 {
		iosObjcMsg1(w.native, iosSelector("setHidden:"), 1)
	}
}
