//go:build rtg && darwin && arm64

package graphics

const appKit = "/System/Library/Frameworks/AppKit.framework/AppKit"
const openGL = "/System/Library/Frameworks/OpenGL.framework/OpenGL"
const objc = "/usr/lib/libobjc.A.dylib"

// rtg:linkstatic /System/Library/Frameworks/AppKit.framework/AppKit,NSApplicationLoad
func nsApplicationLoad() int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_getClass
func objcGetClass(name string) int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,sel_registerName
func selRegisterName(name string) int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsg0(object, selector int) int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsgFloat0(object, selector int) float64 { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsgPointX(object, selector int) float64 { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsgPointY(object, selector int) float64 { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsgRectWidth(object, selector int) float64 { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsgRectHeight(object, selector int) float64 { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsgFloat1(object, selector int, value float64) int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsg1(object, selector, a int) int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsg2(object, selector, a, b int) int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsg4(object, selector, a, b, c, d int) int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsg5(object, selector, a, b, c, d, e int) int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsgBytes(object, selector int, value []byte) int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsgBytes3(object, selector int, value []byte, maximum, encoding int) int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsgInts(object, selector int, value []int32) int { return 0 }

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsgRect(object, selector, x, y, width, height, style, backing int) int {
	return 0
}

// rtg:linkstatic /usr/lib/libobjc.A.dylib,objc_msgSend
func objcMsgSize(object, selector, width, height int) int { return 0 }

// rtg:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glViewport
func glViewport(x, y, width, height int) {}

// rtg:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glMatrixMode
func glMatrixMode(mode int) {}

// rtg:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glLoadIdentity
func glLoadIdentity() {}

// rtg:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glDrawBuffer
func glDrawBuffer(mode int) {}

// rtg:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glOrtho
func glOrtho(left, right, bottom, top, near, far int) {}

// rtg:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glRasterPos2i
func glRasterPos2i(x, y int) {}

// rtg:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glPixelStorei
func glPixelStorei(name, value int) {}

// rtg:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glPixelZoom
func glPixelZoom(x, y int) {}

// rtg:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glDrawPixels
func glDrawPixels(width, height, format, typ int, pixels []byte) {}

// rtg:linkstatic /System/Library/Frameworks/OpenGL.framework/OpenGL,glFlush
func glFlush() {}

const (
	nsApplicationActivationPolicyRegular = 0
	nsWindowStyleTitled                  = 1
	nsWindowStyleClosable                = 2
	nsWindowStyleMiniaturizable          = 4
	nsWindowStyleResizable               = 8
	nsBackingStoreBuffered               = 2
	nsOpenGLPFAAccelerated               = 73
	nsOpenGLPFADoubleBuffer              = 5
	nsOpenGLPFAColorSize                 = 8
	nsOpenGLProfile                      = 99
	nsOpenGLProfileLegacy                = 0x1000
	glProjection                         = 0x1701
	glModelView                          = 0x1700
	glRGBA                               = 0x1908
	glUnsignedByte                       = 0x1401
	glUnpackAlignment                    = 0x0cf5
	glUnpackRowLength                    = 0x0cf2
	glFrontAndBack                       = 0x0408
	nsUTF8StringEncoding                 = 4
	nsModifierShift                      = 1 << 17
	nsModifierControl                    = 1 << 18
	nsModifierOption                     = 1 << 19
	nsModifierCommand                    = 1 << 20
)

var darwinWindows []*Window

func allocDarwinWindow() *Window {
	w := &Window{active: true}
	darwinWindows = append(darwinWindows, w)
	return w
}

func darwinWindowForNative(native int) *Window {
	for i := 0; i < len(darwinWindows); i++ {
		w := darwinWindows[i]
		if w.active && w.native == native {
			return w
		}
	}
	return nil
}

func forgetDarwinWindow(w *Window) {
	for i := 0; i < len(darwinWindows); i++ {
		if darwinWindows[i] == w {
			darwinWindows[i] = darwinWindows[len(darwinWindows)-1]
			darwinWindows = darwinWindows[:len(darwinWindows)-1]
			return
		}
	}
}

func selector(name string) int { return selRegisterName(name) }

func cocoaString(text string) int {
	bytes := append([]byte(text), 0)
	return objcMsgBytes(objcGetClass("NSString"), selector("stringWithUTF8String:"), bytes)
}

func NewWindow(options WindowOptions) *Window {
	if options.Width <= 0 || options.Height <= 0 {
		return nil
	}
	if nsApplicationLoad() == 0 {
		return nil
	}
	w := allocDarwinWindow()
	if w == nil {
		return nil
	}
	w.width = options.Width
	w.height = options.Height
	w.closed = false
	w.shown = !options.Hidden
	w.focused = false
	w.captured = false
	w.cursor = CursorArrow
	w.events = nil
	for i := 0; i < len(w.timerActive); i++ {
		w.timerActive[i] = false
	}
	w.surface = NewSurface(w.width, w.height)
	w.bottomUp = make([]byte, len(w.surface.Pixels))
	w.app = objcMsg0(objcGetClass("NSApplication"), selector("sharedApplication"))
	if w.app == 0 {
		w.Close()
		return nil
	}
	objcMsg1(w.app, selector("setActivationPolicy:"), nsApplicationActivationPolicyRegular)
	objcMsg0(w.app, selector("finishLaunching"))
	w.pool = objcMsg0(objcGetClass("NSAutoreleasePool"), selector("alloc"))
	w.pool = objcMsg0(w.pool, selector("init"))
	style := nsWindowStyleTitled | nsWindowStyleClosable | nsWindowStyleMiniaturizable | nsWindowStyleResizable
	w.native = objcMsg0(objcGetClass("NSWindow"), selector("alloc"))
	w.native = objcMsgRect(w.native, selector("initWithContentRect:styleMask:backing:defer:"), 100, 100, w.width, w.height, style, nsBackingStoreBuffered)
	if w.native == 0 {
		w.Close()
		return nil
	}
	w.backingScale = int(objcMsgFloat0(w.native, selector("backingScaleFactor")))
	if w.backingScale < 1 {
		w.backingScale = 1
	}
	objcMsg1(w.native, selector("setReleasedWhenClosed:"), 0)
	objcMsg1(w.native, selector("setAcceptsMouseMovedEvents:"), 1)
	w.view = objcMsg0(w.native, selector("contentView"))
	attrs := []int32{nsOpenGLPFAAccelerated, nsOpenGLPFADoubleBuffer, nsOpenGLPFAColorSize, 32, nsOpenGLProfile, nsOpenGLProfileLegacy, 0}
	format := objcMsg0(objcGetClass("NSOpenGLPixelFormat"), selector("alloc"))
	format = objcMsgInts(format, selector("initWithAttributes:"), attrs)
	w.context = objcMsg0(objcGetClass("NSOpenGLContext"), selector("alloc"))
	w.context = objcMsg2(w.context, selector("initWithFormat:shareContext:"), format, 0)
	if format != 0 {
		objcMsg0(format, selector("release"))
	}
	if w.context == 0 {
		w.Close()
		return nil
	}
	objcMsg1(w.context, selector("setView:"), w.view)
	objcMsg0(w.context, selector("makeCurrentContext"))
	w.eventMode = cocoaString("kCFRunLoopDefaultMode")
	w.SetTitle(options.Title)
	if !options.Hidden {
		objcMsg1(w.native, selector("makeKeyAndOrderFront:"), 0)
		objcMsg1(w.app, selector("activateIgnoringOtherApps:"), 1)
	}
	w.queue(Event{Type: EventWindowExpose, Dirty: R(0, 0, Scalar(w.width), Scalar(w.height))})
	return w
}

func (w *Window) SetTitle(title string) bool {
	if w == nil || w.native == 0 {
		return false
	}
	objcMsg1(w.native, selector("setTitle:"), cocoaString(title))
	return true
}

func (w *Window) Show() bool {
	if w == nil || w.closed {
		return false
	}
	objcMsg1(w.native, selector("makeKeyAndOrderFront:"), 0)
	w.shown = true
	return true
}

func (w *Window) Hide() bool {
	if w == nil || w.closed {
		return false
	}
	objcMsg1(w.native, selector("orderOut:"), 0)
	w.shown = false
	return true
}

func (w *Window) SetSize(width, height int) bool {
	if w == nil || w.closed || width <= 0 || height <= 0 {
		return false
	}
	objcMsgSize(w.native, selector("setContentSize:"), width, height)
	w.resizeSurface(width, height)
	return true
}

func (w *Window) resizeSurface(width, height int) {
	if width <= 0 || height <= 0 || (width == w.width && height == w.height) {
		return
	}
	w.width, w.height = width, height
	w.surface.Resize(width, height)
	w.bottomUp = make([]byte, len(w.surface.Pixels))
	if w.context != 0 {
		objcMsg0(w.context, selector("update"))
	}
	w.refreshBackingScale()
	w.queue(Event{Type: EventWindowResize, Dirty: R(0, 0, Scalar(width), Scalar(height))})
	w.queue(Event{Type: EventWindowExpose, Dirty: R(0, 0, Scalar(width), Scalar(height))})
}

func (w *Window) refreshBackingScale() {
	if w == nil || w.native == 0 {
		return
	}
	scale := int(objcMsgFloat0(w.native, selector("backingScaleFactor")))
	if scale >= 1 {
		w.backingScale = scale
	}
}

func (w *Window) RequestRepaint(rect Rect) {
	if w != nil && !w.closed {
		w.queue(Event{Type: EventWindowExpose, Dirty: rect})
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
	name := "arrowCursor"
	if cursor == CursorIBeam {
		name = "IBeamCursor"
	}
	if cursor == CursorCrosshair {
		name = "crosshairCursor"
	}
	if cursor == CursorPointingHand {
		name = "pointingHandCursor"
	}
	if cursor == CursorResizeHorizontal {
		name = "resizeLeftRightCursor"
	}
	if cursor == CursorResizeVertical {
		name = "resizeUpDownCursor"
	}
	value := objcMsg0(objcGetClass("NSCursor"), selector(name))
	if value == 0 {
		return false
	}
	objcMsg0(value, selector("set"))
	w.cursor = cursor
	return true
}

func darwinNow() Scalar {
	return objcMsgFloat0(objcGetClass("NSDate"), selector("timeIntervalSinceReferenceDate"))
}

func (w *Window) SetTimer(id int, seconds Scalar) bool {
	if w == nil || w.closed || id == 0 || seconds < 0.0 {
		return false
	}
	slot := -1
	for i := 0; i < len(w.timerActive); i++ {
		if w.timerActive[i] && w.timerIDs[i] == id {
			slot = i
		}
		if slot < 0 && !w.timerActive[i] {
			slot = i
		}
	}
	if slot < 0 {
		return false
	}
	w.timerIDs[slot] = id
	w.timerDeadlines[slot] = darwinNow() + seconds
	w.timerActive[slot] = true
	return true
}

func (w *Window) CancelTimer(id int) {
	if w == nil {
		return
	}
	for i := 0; i < len(w.timerActive); i++ {
		if w.timerActive[i] && w.timerIDs[i] == id {
			w.timerActive[i] = false
		}
	}
}

func (w *Window) queueExpiredTimer() {
	now := darwinNow()
	for i := 0; i < len(w.timerActive); i++ {
		if w.timerActive[i] && w.timerDeadlines[i] <= now {
			id := w.timerIDs[i]
			w.timerActive[i] = false
			w.queue(Event{Type: EventTimer, TimerID: id})
			return
		}
	}
}

func eventModifiers(event int) Modifiers {
	flags := objcMsg0(event, selector("modifierFlags"))
	mods := Modifiers(0)
	if flags&nsModifierShift != 0 {
		mods = mods | ModifierShift
	}
	if flags&nsModifierControl != 0 {
		mods = mods | ModifierControl
	}
	if flags&nsModifierOption != 0 {
		mods = mods | ModifierAlt
	}
	if flags&nsModifierCommand != 0 {
		mods = mods | ModifierCommand
	}
	return mods
}

func cocoaUTF8(value int) string {
	if value == 0 {
		return ""
	}
	length := objcMsg1(value, selector("lengthOfBytesUsingEncoding:"), nsUTF8StringEncoding)
	if length < 0 {
		return ""
	}
	bytes := make([]byte, length+1)
	if objcMsgBytes3(value, selector("getCString:maxLength:encoding:"), bytes, len(bytes), nsUTF8StringEncoding) == 0 {
		return ""
	}
	actual := 0
	for actual < len(bytes) && bytes[actual] != 0 {
		actual++
	}
	return string(bytes[:actual])
}

func (w *Window) dispatchNativeEvent(event int) {
	typ := objcMsg0(event, selector("type"))
	mods := eventModifiers(event)
	if typ == 10 || typ == 11 {
		eventType := EventKeyDown
		if typ == 11 {
			eventType = EventKeyUp
		}
		w.queue(Event{Type: eventType, Key: objcMsg0(event, selector("keyCode")), Modifiers: mods, Repeat: objcMsg0(event, selector("isARepeat")) != 0})
		if typ == 10 {
			text := cocoaUTF8(objcMsg0(event, selector("characters")))
			if text != "" {
				w.queue(Event{Type: EventTextInput, Text: text, Modifiers: mods})
			}
		}
		return
	}
	if typ == 1 || typ == 2 || typ == 3 || typ == 4 || typ == 5 || typ == 6 || typ == 7 || typ == 22 || typ == 25 || typ == 26 || typ == 27 {
		x := objcMsgPointX(event, selector("locationInWindow"))
		y := Scalar(w.height) - objcMsgPointY(event, selector("locationInWindow"))
		button := objcMsg0(event, selector("buttonNumber")) + 1
		if typ == 1 || typ == 3 || typ == 25 {
			w.queue(Event{Type: EventPointerDown, X: x, Y: y, Button: button, Modifiers: mods})
			return
		}
		if typ == 2 || typ == 4 || typ == 26 {
			w.queue(Event{Type: EventPointerUp, X: x, Y: y, Button: button, Modifiers: mods})
			return
		}
		if typ == 22 {
			w.queue(Event{Type: EventPointerWheel, X: x, Y: y, WheelX: objcMsgFloat0(event, selector("scrollingDeltaX")), WheelY: objcMsgFloat0(event, selector("scrollingDeltaY")), Modifiers: mods})
			return
		}
		inside := x >= 0.0 && y >= 0.0 && x < Scalar(w.width) && y < Scalar(w.height)
		if !inside && w.pointerInside && !w.captured {
			w.pointerInside = false
			w.queue(Event{Type: EventPointerLeave, X: x, Y: y, Modifiers: mods})
			return
		}
		if inside {
			w.pointerInside = true
		}
		w.queue(Event{Type: EventPointerMove, X: x, Y: y, Modifiers: mods})
	}
}

func (w *Window) syncWindowState() {
	if objcMsg0(w.native, selector("isVisible")) == 0 && w.shown {
		w.closed = true
		w.queue(Event{Type: EventWindowClose})
		return
	}
	focused := objcMsg0(w.native, selector("isKeyWindow")) != 0
	if focused != w.focused {
		w.focused = focused
		if focused {
			w.queue(Event{Type: EventWindowFocusGained})
		} else {
			w.queue(Event{Type: EventWindowFocusLost})
		}
	}
	width := int(objcMsgRectWidth(w.view, selector("bounds")))
	height := int(objcMsgRectHeight(w.view, selector("bounds")))
	w.resizeSurface(width, height)
	if w.pointerInside && !w.captured {
		x := objcMsgPointX(w.native, selector("mouseLocationOutsideOfEventStream"))
		y := Scalar(w.height) - objcMsgPointY(w.native, selector("mouseLocationOutsideOfEventStream"))
		if x < 0.0 || y < 0.0 || x >= Scalar(w.width) || y >= Scalar(w.height) {
			w.pointerInside = false
			w.queue(Event{Type: EventPointerLeave, X: x, Y: y})
		}
	}
}

func (w *Window) pollNative(date int) {
	for {
		event := objcMsg4(w.app, selector("nextEventMatchingMask:untilDate:inMode:dequeue:"), -1, date, w.eventMode, 1)
		if event == 0 {
			break
		}
		native := objcMsg0(event, selector("window"))
		target := darwinWindowForNative(native)
		if target == nil {
			target = w
		}
		target.dispatchNativeEvent(event)
		objcMsg1(w.app, selector("sendEvent:"), event)
		date = objcMsg0(objcGetClass("NSDate"), selector("distantPast"))
	}
	w.syncWindowState()
	w.queueExpiredTimer()
}

func (w *Window) Poll() (Event, bool) {
	if w == nil {
		return Event{}, false
	}
	if event, ok := w.nextQueuedEvent(); ok {
		return event, true
	}
	if w.closed {
		return Event{}, false
	}
	w.pollNative(objcMsg0(objcGetClass("NSDate"), selector("distantPast")))
	return w.nextQueuedEvent()
}

func (w *Window) Wait() (Event, bool) {
	if w == nil {
		return Event{}, false
	}
	for !w.closed {
		if event, ok := w.nextQueuedEvent(); ok {
			return event, true
		}
		w.queueExpiredTimer()
		if event, ok := w.nextQueuedEvent(); ok {
			return event, true
		}
		date := objcMsg0(objcGetClass("NSDate"), selector("distantFuture"))
		now := darwinNow()
		delay := Scalar(-1.0)
		for i := 0; i < len(w.timerActive); i++ {
			if w.timerActive[i] {
				candidate := w.timerDeadlines[i] - now
				if candidate < 0.0 {
					candidate = 0.0
				}
				if delay < 0.0 || candidate < delay {
					delay = candidate
				}
			}
		}
		if delay >= 0.0 {
			date = objcMsgFloat1(objcGetClass("NSDate"), selector("dateWithTimeIntervalSinceNow:"), delay)
		}
		w.pollNative(date)
	}
	return Event{}, false
}

func SetClipboardText(text string) bool {
	pasteboard := objcMsg0(objcGetClass("NSPasteboard"), selector("generalPasteboard"))
	if pasteboard == 0 {
		return false
	}
	objcMsg0(pasteboard, selector("clearContents"))
	typeName := cocoaString("public.utf8-plain-text")
	return objcMsg2(pasteboard, selector("setString:forType:"), cocoaString(text), typeName) != 0
}

func ClipboardText() (string, bool) {
	pasteboard := objcMsg0(objcGetClass("NSPasteboard"), selector("generalPasteboard"))
	if pasteboard == 0 {
		return "", false
	}
	value := objcMsg1(pasteboard, selector("stringForType:"), cocoaString("public.utf8-plain-text"))
	if value == 0 {
		return "", false
	}
	return cocoaUTF8(value), true
}

func (w *Window) Present() bool {
	if w == nil || w.context == 0 {
		return false
	}
	if !w.surface.dirtyValid {
		return true
	}
	w.refreshBackingScale()
	dirty := intersectPixelRect(pixelRect{maxX: w.width, maxY: w.height}, w.surface.dirty)
	if dirty.maxX <= dirty.minX || dirty.maxY <= dirty.minY {
		w.surface.ResetDirty()
		return true
	}
	row := w.surface.Stride
	for y := dirty.minY; y < dirty.maxY; y++ {
		src := y * row
		dst := (w.height - y - 1) * row
		for x := dirty.minX * 4; x < dirty.maxX*4; x++ {
			w.bottomUp[dst+x] = w.surface.Pixels[src+x]
		}
	}
	objcMsg0(w.context, selector("makeCurrentContext"))
	glViewport(0, 0, w.width*w.backingScale, w.height*w.backingScale)
	glMatrixMode(glProjection)
	glLoadIdentity()
	glOrtho(0, w.width, 0, w.height, -1, 1)
	glMatrixMode(glModelView)
	glLoadIdentity()
	glDrawBuffer(glFrontAndBack)
	glRasterPos2i(dirty.minX, w.height-dirty.maxY)
	glPixelZoom(w.backingScale, w.backingScale)
	glPixelStorei(glUnpackAlignment, 1)
	glPixelStorei(glUnpackRowLength, w.width)
	start := (w.height-dirty.maxY)*row + dirty.minX*4
	glDrawPixels(dirty.maxX-dirty.minX, dirty.maxY-dirty.minY, glRGBA, glUnsignedByte, w.bottomUp[start:])
	glPixelStorei(glUnpackRowLength, 0)
	glFlush()
	objcMsg0(w.context, selector("flushBuffer"))
	w.surface.ResetDirty()
	return true
}

func (w *Window) Close() {
	if w == nil || w.closed {
		return
	}
	w.closed = true
	w.shown = false
	if w.context != 0 {
		objcMsg0(w.context, selector("clearDrawable"))
		objcMsg0(w.context, selector("release"))
		w.context = 0
	}
	if w.native != 0 {
		objcMsg0(w.native, selector("close"))
		objcMsg0(w.native, selector("release"))
		w.native = 0
	}
	if w.pool != 0 {
		objcMsg0(w.pool, selector("drain"))
		w.pool = 0
	}
	w.active = false
	forgetDarwinWindow(w)
}
