//go:build renvo && windows && (amd64 || 386 || arm64)

package graphics

// The Windows backend deliberately uses the original WGL/OpenGL 1.1 entry
// points exported by opengl32.dll. Rendering remains backend-neutral and is
// performed into Surface.Pixels; WGL is only the presentation layer.

// renvo:linkstatic kernel32.dll,GetProcAddress
func windowsGetProcAddress(module int, name *byte) int { return 0 }

// renvo:linkstatic kernel32.dll,GetLastError
func windowsGetLastError() int { return 0 }

// renvo:linkstatic kernel32.dll,GlobalAlloc
func windowsGlobalAlloc(flags, size int) int { return 0 }

// renvo:linkstatic kernel32.dll,GlobalFree
func windowsGlobalFree(memory int) int { return 0 }

// renvo:linkstatic kernel32.dll,GlobalLock
func windowsGlobalLock(memory int) int { return 0 }

// renvo:linkstatic kernel32.dll,GlobalUnlock
func windowsGlobalUnlock(memory int) int { return 0 }

// renvo:linkstatic kernel32.dll,GlobalSize
func windowsGlobalSize(memory int) int { return 0 }

// RtlMoveMemory is exported by Kernel32 on both Windows 98 and NT-family
// systems. Importing the native NT implementation directly prevents the image
// loader from starting the program on Windows 9x, which has no ntdll.dll.
// renvo:linkstatic kernel32.dll,RtlMoveMemory
func windowsMoveMemory(destination int, source *byte, size int) {}

// renvo:linkstatic kernel32.dll,RtlMoveMemory
func windowsReadMemory(destination *byte, source int, size int) {}

// renvo:linkstatic user32.dll,DestroyWindow
func windowsDestroyWindow(window int) int { return 0 }

// renvo:linkstatic user32.dll,ShowWindow
func windowsShowWindow(window, command int) int { return 0 }

// renvo:linkstatic user32.dll,UpdateWindow
func windowsUpdateWindow(window int) int { return 0 }

// renvo:linkstatic user32.dll,AdjustWindowRectEx
func windowsAdjustWindowRectRaw(rect *byte, style, menu, exStyle int) int { return 0 }

// renvo:linkstatic user32.dll,SetWindowPos
func windowsSetWindowPos(window, after, x, y, width, height, flags int) int { return 0 }

// renvo:linkstatic user32.dll,InvalidateRect
func windowsInvalidateRectRaw(window int, rect *byte, erase int) int { return 0 }

// renvo:linkstatic user32.dll,GetUpdateRect
func windowsGetUpdateRectRaw(window int, rect *byte, erase int) int { return 0 }

// renvo:linkstatic user32.dll,ValidateRect
func windowsValidateRectRaw(window int, rect *byte) int { return 0 }

// renvo:linkstatic user32.dll,TranslateMessage
func windowsTranslateMessageRaw(message *byte) int { return 0 }

// renvo:linkstatic user32.dll,GetKeyState
func windowsGetKeyState(key int) int { return 0 }

// renvo:linkstatic user32.dll,TrackMouseEvent
func windowsBeginTrackMouseEventRaw(event *byte) int { return 0 }

// renvo:linkstatic user32.dll,ScreenToClient
func windowsScreenToClientRaw(window int, point *byte) int { return 0 }

// renvo:linkstatic user32.dll,SetCapture
func windowsSetCapture(window int) int { return 0 }

// renvo:linkstatic user32.dll,ReleaseCapture
func windowsReleaseCapture() int { return 0 }

// renvo:linkstatic user32.dll,SetCursor
func windowsSetCursor(cursor int) int { return 0 }

// renvo:linkstatic user32.dll,SetTimer
func windowsSetTimer(window, id, milliseconds, callback int) int { return 0 }

// renvo:linkstatic user32.dll,KillTimer
func windowsKillTimer(window, id int) int { return 0 }

// renvo:linkstatic user32.dll,OpenClipboard
func windowsOpenClipboard(owner int) int { return 0 }

// renvo:linkstatic user32.dll,CloseClipboard
func windowsCloseClipboard() int { return 0 }

// renvo:linkstatic user32.dll,EmptyClipboard
func windowsEmptyClipboard() int { return 0 }

// renvo:linkstatic user32.dll,GetClipboardData
func windowsGetClipboardData(format int) int { return 0 }

// renvo:linkstatic user32.dll,SetClipboardData
func windowsSetClipboardData(format, memory int) int { return 0 }

// renvo:linkstatic user32.dll,GetDC
func windowsGetDC(window int) int { return 0 }

// renvo:linkstatic user32.dll,ReleaseDC
func windowsReleaseDC(window, device int) int { return 0 }

// renvo:linkstatic gdi32.dll,ChoosePixelFormat
func windowsChoosePixelFormat(device int, format *byte) int { return 0 }

// renvo:linkstatic gdi32.dll,SetPixelFormat
func windowsSetPixelFormat(device, index int, format *byte) int { return 0 }

// renvo:linkstatic gdi32.dll,SwapBuffers
func windowsSwapBuffers(device int) int { return 0 }

// renvo:linkstatic opengl32.dll,wglCreateContext
func windowsWGLCreateContext(device int) int { return 0 }

// renvo:linkstatic opengl32.dll,wglMakeCurrent
func windowsWGLMakeCurrent(device, context int) int { return 0 }

// renvo:linkstatic opengl32.dll,wglDeleteContext
func windowsWGLDeleteContext(context int) int { return 0 }

// renvo:linkstatic opengl32.dll,glViewport
func glViewport(x, y, width, height int) {}

// renvo:linkstatic opengl32.dll,glMatrixMode
func glMatrixMode(mode int) {}

// renvo:linkstatic opengl32.dll,glLoadIdentity
func glLoadIdentity() {}

// renvo:linkstatic opengl32.dll,glDrawBuffer
func glDrawBuffer(mode int) {}

// renvo:linkstatic opengl32.dll,glRasterPos2i
func glRasterPos2i(x, y int) {}

// renvo:linkstatic opengl32.dll,glPixelStorei
func glPixelStorei(name, value int) {}

// renvo:linkstatic opengl32.dll,glDrawPixels
func glDrawPixels(width, height, format, typ int, pixels []byte) {}

// renvo:linkstatic opengl32.dll,glReadBuffer
func glReadBuffer(mode int) {}

// renvo:linkstatic opengl32.dll,glReadPixels
func glReadPixels(x, y, width, height, format, typ int, pixels []byte) {}

// renvo:linkstatic opengl32.dll,glFinish
func glFinish() {}

type windowsRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type windowsPoint struct {
	X int32
	Y int32
}

type windowsMessage struct {
	Window  int
	Message uint32
	Padding uint32
	WParam  int
	LParam  int
	Time    uint32
	Point   windowsPoint
	Private uint32
}

type windowsTrackMouseEvent struct {
	Size      uint32
	Flags     uint32
	Window    int
	HoverTime uint32
	Padding   uint32
}

const (
	windowsClassOwnDC          = 0x0020
	windowsClassHRedraw        = 0x0002
	windowsClassVRedraw        = 0x0001
	windowsStyleOverlapped     = 0x00cf0000
	windowsStyleClipChildren   = 0x02000000
	windowsStyleClipSiblings   = 0x04000000
	windowsUseDefault          = -2147483648
	windowsShowHide            = 0
	windowsShowNormal          = 5
	windowsSetPosNoMove        = 0x0002
	windowsSetPosNoZOrder      = 0x0004
	windowsPeekRemove          = 1
	windowsTrackLeave          = 2
	windowsFormatDrawToWindow  = 0x00000004
	windowsFormatSupportOpenGL = 0x00000020
	windowsFormatDoubleBuffer  = 0x00000001
	windowsFormatRGBA          = 0
	windowsMainPlane           = 0
	windowsGlobalMoveable      = 0x0002

	windowsMessageSize          = 0x0005
	windowsMessageSetFocus      = 0x0007
	windowsMessageKillFocus     = 0x0008
	windowsMessagePaint         = 0x000f
	windowsMessageClose         = 0x0010
	windowsMessageDestroy       = 0x0002
	windowsMessageQuit          = 0x0012
	windowsMessageSetCursor     = 0x0020
	windowsMessageTimer         = 0x0113
	windowsMessageKeyDown       = 0x0100
	windowsMessageKeyUp         = 0x0101
	windowsMessageCharacter     = 0x0102
	windowsMessageSystemKeyDown = 0x0104
	windowsMessageSystemKeyUp   = 0x0105
	windowsMessageMouseMove     = 0x0200
	windowsMessageLeftDown      = 0x0201
	windowsMessageLeftUp        = 0x0202
	windowsMessageRightDown     = 0x0204
	windowsMessageRightUp       = 0x0205
	windowsMessageMiddleDown    = 0x0207
	windowsMessageMiddleUp      = 0x0208
	windowsMessageMouseWheel    = 0x020a
	windowsMessageMouseHWheel   = 0x020e
	windowsMessageMouseLeave    = 0x02a3

	windowsVirtualShift   = 0x10
	windowsVirtualControl = 0x11
	windowsVirtualAlt     = 0x12

	windowsCursorArrow  = 32512
	windowsCursorIBeam  = 32513
	windowsCursorCross  = 32515
	windowsCursorSizeWE = 32644
	windowsCursorSizeNS = 32645
	windowsCursorHand   = 32649

	glProjection      = 0x1701
	glModelView       = 0x1700
	glRGBA            = 0x1908
	glUnsignedByte    = 0x1401
	glUnpackAlignment = 0x0cf5
	glUnpackRowLength = 0x0cf2
	glPackAlignment   = 0x0d05
	glFront           = 0x0404
	glBack            = 0x0405
)

var windowsClassName []byte
var windowsClassReady bool
var windowsClassAtom int
var windowsClassInstance int
var windowsWindows []*Window

func windowsASCII(text string) []byte {
	bytes := make([]byte, len(text)+1)
	for i := 0; i < len(text); i++ {
		bytes[i] = text[i]
	}
	return bytes
}

func windowsPut16(data []byte, offset, value int) {
	data[offset] = byte(value)
	data[offset+1] = byte(value >> 8)
}

func windowsPut32(data []byte, offset, value int) {
	data[offset] = byte(value)
	data[offset+1] = byte(value >> 8)
	data[offset+2] = byte(value >> 16)
	data[offset+3] = byte(value >> 24)
}

func windowsPutPointer(data []byte, offset, value int) {
	windowsPut32(data, offset, value)
	if windowsPointerSize == 8 {
		windowsPut32(data, offset+4, value>>32)
	}
}

func windowsGet32(data []byte, offset int) int {
	return int(data[offset]) | int(data[offset+1])<<8 | int(data[offset+2])<<16 | int(data[offset+3])<<24
}

func windowsGetSigned32(data []byte, offset int) int {
	value := windowsGet32(data, offset)
	if value >= 0x80000000 {
		value -= 0x100000000
	}
	return value
}

func windowsGetPointer(data []byte, offset int) int {
	value := windowsGet32(data, offset)
	if windowsPointerSize == 8 {
		value = value | windowsGet32(data, offset+4)<<32
	}
	return value
}

func windowsRegisterClass(style, windowProc, instance, cursor, className int) int {
	data := make([]byte, windowsWindowClassSize)
	windowsPut32(data, 0, style)
	windowsPutPointer(data, windowsWindowClassProcOffset, windowProc)
	windowsPutPointer(data, windowsWindowClassInstanceOffset, instance)
	windowsPutPointer(data, windowsWindowClassCursorOffset, cursor)
	windowsPutPointer(data, windowsWindowClassNameOffset, className)
	// RegisterClass returns ATOM, a 16-bit value. The upper half of EAX is not
	// part of the ABI result and is left nonzero by User32 on Windows XP.
	return windowsRegisterClassRaw(&data[0]) & 0xffff
}

func windowsRectData(rect *windowsRect) []byte {
	data := make([]byte, 16)
	if rect != nil {
		windowsPut32(data, 0, int(rect.Left))
		windowsPut32(data, 4, int(rect.Top))
		windowsPut32(data, 8, int(rect.Right))
		windowsPut32(data, 12, int(rect.Bottom))
	}
	return data
}

func windowsReadRect(data []byte, rect *windowsRect) {
	if rect == nil {
		return
	}
	rect.Left = int32(windowsGetSigned32(data, 0))
	rect.Top = int32(windowsGetSigned32(data, 4))
	rect.Right = int32(windowsGetSigned32(data, 8))
	rect.Bottom = int32(windowsGetSigned32(data, 12))
}

func windowsAdjustWindowRect(rect *windowsRect, style, menu, exStyle int) int {
	data := windowsRectData(rect)
	result := windowsAdjustWindowRectRaw(&data[0], style, menu, exStyle)
	windowsReadRect(data, rect)
	return result
}

func windowsInvalidateRect(window int, rect *windowsRect, erase int) int {
	if rect == nil {
		return windowsInvalidateRectRaw(window, nil, erase)
	}
	data := windowsRectData(rect)
	return windowsInvalidateRectRaw(window, &data[0], erase)
}

func windowsGetUpdateRect(window int, rect *windowsRect, erase int) int {
	if rect == nil {
		return windowsGetUpdateRectRaw(window, nil, erase)
	}
	data := windowsRectData(rect)
	result := windowsGetUpdateRectRaw(window, &data[0], erase)
	windowsReadRect(data, rect)
	return result
}

func windowsValidateRect(window int, rect *windowsRect) int {
	if rect == nil {
		return windowsValidateRectRaw(window, nil)
	}
	data := windowsRectData(rect)
	return windowsValidateRectRaw(window, &data[0])
}

func windowsMessageData(message *windowsMessage) []byte {
	data := make([]byte, windowsMessageStructSize)
	windowsPutPointer(data, windowsMessageWindowOffset, message.Window)
	windowsPut32(data, windowsMessageKindOffset, int(message.Message))
	windowsPutPointer(data, windowsMessageWParamOffset, message.WParam)
	windowsPutPointer(data, windowsMessageLParamOffset, message.LParam)
	windowsPut32(data, windowsMessageTimeOffset, int(message.Time))
	windowsPut32(data, windowsMessagePointXOffset, int(message.Point.X))
	windowsPut32(data, windowsMessagePointYOffset, int(message.Point.Y))
	windowsPut32(data, windowsMessagePrivateOffset, int(message.Private))
	return data
}

func windowsReadMessage(data []byte, message *windowsMessage) {
	message.Window = windowsGetPointer(data, windowsMessageWindowOffset)
	message.Message = uint32(windowsGet32(data, windowsMessageKindOffset))
	message.WParam = windowsGetPointer(data, windowsMessageWParamOffset)
	message.LParam = windowsGetPointer(data, windowsMessageLParamOffset)
	message.Time = uint32(windowsGet32(data, windowsMessageTimeOffset))
	message.Point.X = int32(windowsGetSigned32(data, windowsMessagePointXOffset))
	message.Point.Y = int32(windowsGetSigned32(data, windowsMessagePointYOffset))
	message.Private = uint32(windowsGet32(data, windowsMessagePrivateOffset))
}

func windowsPeekMessage(message *windowsMessage, window, first, last, remove int) int {
	data := windowsMessageData(message)
	result := windowsPeekMessageRaw(&data[0], window, first, last, remove)
	windowsReadMessage(data, message)
	return result
}

func windowsGetMessage(message *windowsMessage, window, first, last int) int {
	data := windowsMessageData(message)
	result := windowsGetMessageRaw(&data[0], window, first, last)
	windowsReadMessage(data, message)
	return result
}

func windowsTranslateMessage(message *windowsMessage) int {
	data := windowsMessageData(message)
	return windowsTranslateMessageRaw(&data[0])
}

func windowsDispatchMessage(message *windowsMessage) int {
	data := windowsMessageData(message)
	return windowsDispatchMessageRaw(&data[0])
}

func windowsBeginTrackMouseEvent(event *windowsTrackMouseEvent) int {
	data := make([]byte, windowsTrackMouseEventSize)
	windowsPut32(data, 0, int(event.Size))
	windowsPut32(data, 4, int(event.Flags))
	windowsPutPointer(data, 8, event.Window)
	windowsPut32(data, windowsTrackMouseEventHoverTimeOffset, int(event.HoverTime))
	return windowsBeginTrackMouseEventRaw(&data[0])
}

func windowsScreenToClient(window int, point *windowsPoint) int {
	data := make([]byte, 8)
	windowsPut32(data, 0, int(point.X))
	windowsPut32(data, 4, int(point.Y))
	result := windowsScreenToClientRaw(window, &data[0])
	point.X = int32(windowsGetSigned32(data, 0))
	point.Y = int32(windowsGetSigned32(data, 4))
	return result
}

func windowsDecodeUTF8(text string, index int) (int, int) {
	first := int(text[index])
	if first < 0x80 {
		return first, index + 1
	}
	if first&0xe0 == 0xc0 && index+1 < len(text) {
		return (first&0x1f)<<6 | int(text[index+1])&0x3f, index + 2
	}
	if first&0xf0 == 0xe0 && index+2 < len(text) {
		return (first&0x0f)<<12 | (int(text[index+1])&0x3f)<<6 | int(text[index+2])&0x3f, index + 3
	}
	if first&0xf8 == 0xf0 && index+3 < len(text) {
		return (first&7)<<18 | (int(text[index+1])&0x3f)<<12 | (int(text[index+2])&0x3f)<<6 | int(text[index+3])&0x3f, index + 4
	}
	return 0xfffd, index + 1
}

func windowsAppendUTF16(out []byte, value int) []byte {
	if value < 0 || value > 0x10ffff || (value >= 0xd800 && value <= 0xdfff) {
		value = 0xfffd
	}
	if value >= 0x10000 {
		value -= 0x10000
		high := 0xd800 + value/0x400
		low := 0xdc00 + value%0x400
		out = append(out, byte(high))
		out = append(out, byte(high>>8))
		out = append(out, byte(low))
		out = append(out, byte(low>>8))
		return out
	}
	out = append(out, byte(value))
	return append(out, byte(value>>8))
}

func windowsUTF16(text string) []byte {
	out := make([]byte, 0, len(text)*2+2)
	for i := 0; i < len(text); {
		value, next := windowsDecodeUTF8(text, i)
		out = windowsAppendUTF16(out, value)
		i = next
	}
	out = append(out, 0)
	return append(out, 0)
}

func windowsNativeBytes(data []byte) (int, int) {
	memory := windowsGlobalAlloc(0, len(data))
	if memory == 0 {
		return 0, 0
	}
	address := windowsGlobalLock(memory)
	if address == 0 {
		windowsGlobalFree(memory)
		return 0, 0
	}
	windowsMoveMemory(address, &data[0], len(data))
	return memory, address
}

func windowsAppendUTF8(out []byte, value int) []byte {
	if value < 0 || value > 0x10ffff || (value >= 0xd800 && value <= 0xdfff) {
		value = 0xfffd
	}
	if value < 0x80 {
		return append(out, byte(value))
	}
	if value < 0x800 {
		out = append(out, byte(0xc0|value>>6))
		return append(out, byte(0x80|value&0x3f))
	}
	if value < 0x10000 {
		out = append(out, byte(0xe0|value>>12))
		out = append(out, byte(0x80|value>>6&0x3f))
		return append(out, byte(0x80|value&0x3f))
	}
	out = append(out, byte(0xf0|value>>18))
	out = append(out, byte(0x80|value>>12&0x3f))
	out = append(out, byte(0x80|value>>6&0x3f))
	return append(out, byte(0x80|value&0x3f))
}

func windowsUTF16BytesToString(bytes []byte) string {
	out := make([]byte, 0, len(bytes)/2)
	pending := 0
	for i := 0; i+1 < len(bytes); i += 2 {
		unit := int(bytes[i]) | int(bytes[i+1])<<8
		if unit == 0 {
			break
		}
		if unit >= 0xd800 && unit <= 0xdbff {
			if pending != 0 {
				out = windowsAppendUTF8(out, 0xfffd)
			}
			pending = unit
			continue
		}
		if unit >= 0xdc00 && unit <= 0xdfff && pending != 0 {
			value := 0x10000 + (pending-0xd800)*0x400 + unit - 0xdc00
			out = windowsAppendUTF8(out, value)
			pending = 0
			continue
		}
		if pending != 0 {
			out = windowsAppendUTF8(out, 0xfffd)
			pending = 0
		}
		out = windowsAppendUTF8(out, unit)
	}
	if pending != 0 {
		out = windowsAppendUTF8(out, 0xfffd)
	}
	return string(out)
}

func windowsRegisterGraphicsClass() bool {
	if windowsClassReady {
		return true
	}
	if len(windowsClassName) == 0 {
		windowsClassName = windowsNativeString("RenvoGraphicsWindow")
	}
	instance := windowsGetModuleHandle(nil)
	if instance == 0 {
		setLastWindowError("GetModuleHandle failed", windowsGetLastError())
		return false
	}
	user32 := windowsNativeString("user32.dll")
	module := windowsGetModuleHandle(&user32[0])
	procName := windowsASCII(windowsDefWindowProcName)
	windowProc := windowsGetProcAddress(module, &procName[0])
	if module == 0 || windowProc == 0 {
		setLastWindowError("DefWindowProc lookup failed", windowsGetLastError())
		return false
	}
	classNameMemory, classNameAddress := windowsNativeBytes(windowsClassName)
	if classNameMemory == 0 {
		setLastWindowError("window class name allocation failed", windowsGetLastError())
		return false
	}
	registered := windowsRegisterClass(windowsClassOwnDC|windowsClassHRedraw|windowsClassVRedraw, windowProc, instance, windowsLoadCursor(0, windowsCursorArrow), classNameAddress)
	if registered == 0 {
		code := windowsGetLastError()
		windowsGlobalUnlock(classNameMemory)
		windowsGlobalFree(classNameMemory)
		setLastWindowError("RegisterClass failed", code)
		return false
	}
	windowsGlobalUnlock(classNameMemory)
	windowsGlobalFree(classNameMemory)
	windowsClassAtom = registered
	windowsClassInstance = instance
	windowsClassReady = true
	return true
}

func allocWindowsWindow() *Window {
	w := &Window{active: true}
	windowsWindows = append(windowsWindows, w)
	return w
}

func windowsWindowForNative(native int) *Window {
	for i := 0; i < len(windowsWindows); i++ {
		w := windowsWindows[i]
		if w.active && w.native == native {
			return w
		}
	}
	return nil
}

func forgetWindowsWindow(window *Window) {
	for i := 0; i < len(windowsWindows); i++ {
		if windowsWindows[i] == window {
			windowsWindows[i] = windowsWindows[len(windowsWindows)-1]
			windowsWindows = windowsWindows[:len(windowsWindows)-1]
			return
		}
	}
}

func NewWindow(options WindowOptions) *Window {
	clearLastWindowError()
	if options.Width <= 0 || options.Height <= 0 {
		setLastWindowError("window dimensions must be positive", 0)
		return nil
	}
	if !windowsRegisterGraphicsClass() {
		return nil
	}
	w := allocWindowsWindow()
	w.width = options.Width
	w.height = options.Height
	w.instance = windowsClassInstance
	w.cursor = CursorArrow
	w.shown = !options.Hidden
	w.surface = NewSurface(w.width, w.height)
	w.bottomUp = make([]byte, len(w.surface.Pixels))
	style := windowsStyleOverlapped | windowsStyleClipChildren | windowsStyleClipSiblings
	rect := windowsRect{Right: int32(w.width), Bottom: int32(w.height)}
	if windowsAdjustWindowRect(&rect, style, 0, 0) == 0 {
		setLastWindowError("AdjustWindowRectEx failed", windowsGetLastError())
		w.Close()
		return nil
	}
	titleMemory, titleAddress := windowsNativeBytes(windowsNativeString(options.Title))
	if titleMemory == 0 {
		setLastWindowError("window title allocation failed", windowsGetLastError())
		w.Close()
		return nil
	}
	w.native = windowsCreateWindow(0, windowsClassAtom, titleAddress, style, windowsUseDefault, windowsUseDefault, int(rect.Right-rect.Left), int(rect.Bottom-rect.Top), 0, 0, w.instance, 0)
	windowsGlobalUnlock(titleMemory)
	windowsGlobalFree(titleMemory)
	if w.native == 0 {
		setLastWindowError("CreateWindowEx failed", windowsGetLastError())
		w.Close()
		return nil
	}
	w.device = windowsGetDC(w.native)
	if w.device == 0 {
		setLastWindowError("GetDC failed", windowsGetLastError())
		w.Close()
		return nil
	}
	format := make([]byte, 40)
	windowsPut16(format, 0, 40)
	windowsPut16(format, 2, 1)
	windowsPut32(format, 4, windowsFormatDrawToWindow|windowsFormatSupportOpenGL|windowsFormatDoubleBuffer)
	format[8] = byte(windowsFormatRGBA)
	format[9] = 32
	format[16] = 8
	format[23] = 24
	format[26] = byte(windowsMainPlane)
	formatIndex := windowsChoosePixelFormat(w.device, &format[0])
	if formatIndex == 0 {
		setLastWindowError("ChoosePixelFormat failed", windowsGetLastError())
		w.Close()
		return nil
	}
	if windowsSetPixelFormat(w.device, formatIndex, &format[0]) == 0 {
		setLastWindowError("SetPixelFormat failed", windowsGetLastError())
		w.Close()
		return nil
	}
	w.context = windowsWGLCreateContext(w.device)
	if w.context == 0 {
		setLastWindowError("wglCreateContext failed", windowsGetLastError())
		w.Close()
		return nil
	}
	if windowsWGLMakeCurrent(w.device, w.context) == 0 {
		setLastWindowError("wglMakeCurrent failed", windowsGetLastError())
		w.Close()
		return nil
	}
	if !options.Hidden {
		windowsShowWindow(w.native, windowsShowNormal)
		windowsUpdateWindow(w.native)
	}
	w.queue(Event{Type: EventWindowExpose, Dirty: R(0, 0, Scalar(w.width), Scalar(w.height))})
	return w
}

func (w *Window) SetTitle(title string) bool {
	if w == nil || w.closed || w.native == 0 {
		return false
	}
	text := windowsNativeString(title)
	return windowsSetWindowText(w.native, &text[0]) != 0
}

func (w *Window) Show() bool {
	if w == nil || w.closed || w.native == 0 {
		return false
	}
	windowsShowWindow(w.native, windowsShowNormal)
	windowsUpdateWindow(w.native)
	w.shown = true
	return true
}

func (w *Window) Hide() bool {
	if w == nil || w.closed || w.native == 0 {
		return false
	}
	windowsShowWindow(w.native, windowsShowHide)
	w.shown = false
	return true
}

func (w *Window) SetSize(width, height int) bool {
	if w == nil || w.closed || width <= 0 || height <= 0 {
		return false
	}
	style := windowsStyleOverlapped | windowsStyleClipChildren | windowsStyleClipSiblings
	rect := windowsRect{Right: int32(width), Bottom: int32(height)}
	if windowsAdjustWindowRect(&rect, style, 0, 0) == 0 {
		return false
	}
	if windowsSetWindowPos(w.native, 0, 0, 0, int(rect.Right-rect.Left), int(rect.Bottom-rect.Top), windowsSetPosNoMove|windowsSetPosNoZOrder) == 0 {
		return false
	}
	w.resizeSurface(width, height)
	return true
}

func (w *Window) resizeSurface(width, height int) {
	if width <= 0 || height <= 0 || (width == w.width && height == w.height) {
		return
	}
	w.width = width
	w.height = height
	w.surface.Resize(width, height)
	w.bottomUp = make([]byte, len(w.surface.Pixels))
	w.queue(Event{Type: EventWindowResize, Dirty: R(0, 0, Scalar(width), Scalar(height))})
	w.queue(Event{Type: EventWindowExpose, Dirty: R(0, 0, Scalar(width), Scalar(height))})
}

func (w *Window) RequestRepaint(rect Rect) {
	if w == nil || w.closed {
		return
	}
	nativeRect := windowsRect{Left: int32(rect.MinX), Top: int32(rect.MinY), Right: int32(rect.MaxX), Bottom: int32(rect.MaxY)}
	windowsInvalidateRect(w.native, &nativeRect, 0)
}

func (w *Window) SetPointerCapture(captured bool) bool {
	if w == nil || w.closed {
		return false
	}
	if captured {
		windowsSetCapture(w.native)
	} else {
		windowsReleaseCapture()
	}
	w.captured = captured
	return true
}

func windowsCursorResource(cursor Cursor) int {
	if cursor == CursorIBeam {
		return windowsCursorIBeam
	}
	if cursor == CursorCrosshair {
		return windowsCursorCross
	}
	if cursor == CursorPointingHand {
		return windowsCursorHand
	}
	if cursor == CursorResizeHorizontal {
		return windowsCursorSizeWE
	}
	if cursor == CursorResizeVertical {
		return windowsCursorSizeNS
	}
	return windowsCursorArrow
}

func (w *Window) SetCursor(cursor Cursor) bool {
	if w == nil || w.closed {
		return false
	}
	nativeCursor := windowsLoadCursor(0, windowsCursorResource(cursor))
	if nativeCursor == 0 {
		return false
	}
	windowsSetCursor(nativeCursor)
	w.cursor = cursor
	return true
}

func (w *Window) SetTimer(id int, seconds Scalar) bool {
	if w == nil || w.closed || id <= 0 || seconds < 0.0 {
		return false
	}
	milliseconds := int(seconds * 1000.0)
	if milliseconds < 1 {
		milliseconds = 1
	}
	if windowsSetTimer(w.native, id, milliseconds, 0) == 0 {
		return false
	}
	for i := 0; i < len(w.timerActive); i++ {
		if w.timerActive[i] && w.timerIDs[i] == id {
			return true
		}
	}
	for i := 0; i < len(w.timerActive); i++ {
		if !w.timerActive[i] {
			w.timerActive[i] = true
			w.timerIDs[i] = id
			return true
		}
	}
	windowsKillTimer(w.native, id)
	return false
}

func (w *Window) CancelTimer(id int) {
	if w == nil {
		return
	}
	windowsKillTimer(w.native, id)
	for i := 0; i < len(w.timerActive); i++ {
		if w.timerActive[i] && w.timerIDs[i] == id {
			w.timerActive[i] = false
		}
	}
}

func windowsModifiers() Modifiers {
	modifiers := Modifiers(0)
	if windowsGetKeyState(windowsVirtualShift)&0x8000 != 0 {
		modifiers = modifiers | ModifierShift
	}
	if windowsGetKeyState(windowsVirtualControl)&0x8000 != 0 {
		modifiers = modifiers | ModifierControl
	}
	if windowsGetKeyState(windowsVirtualAlt)&0x8000 != 0 {
		modifiers = modifiers | ModifierAlt
	}
	return modifiers
}

func windowsSignedWord(value int) int {
	value = value & 0xffff
	if value >= 0x8000 {
		value -= 0x10000
	}
	return value
}

func (w *Window) queueUTF16(unit int, modifiers Modifiers) {
	if unit >= 0xd800 && unit <= 0xdbff {
		if w.pendingUTF16 != 0 {
			bytes := windowsAppendUTF8(nil, 0xfffd)
			w.queue(Event{Type: EventTextInput, Text: string(bytes), Modifiers: modifiers})
		}
		w.pendingUTF16 = unit
		return
	}
	value := unit
	if unit >= 0xdc00 && unit <= 0xdfff && w.pendingUTF16 != 0 {
		value = 0x10000 + (w.pendingUTF16-0xd800)*0x400 + unit - 0xdc00
	} else if w.pendingUTF16 != 0 {
		bytes := windowsAppendUTF8(nil, 0xfffd)
		w.queue(Event{Type: EventTextInput, Text: string(bytes), Modifiers: modifiers})
	}
	w.pendingUTF16 = 0
	bytes := windowsAppendUTF8(nil, value)
	w.queue(Event{Type: EventTextInput, Text: string(bytes), Modifiers: modifiers})
}

func (w *Window) queuePointer(message int, x, y int, modifiers Modifiers) {
	eventType := EventPointerMove
	button := 0
	if message == windowsMessageLeftDown || message == windowsMessageRightDown || message == windowsMessageMiddleDown {
		eventType = EventPointerDown
	}
	if message == windowsMessageLeftUp || message == windowsMessageRightUp || message == windowsMessageMiddleUp {
		eventType = EventPointerUp
	}
	if message == windowsMessageLeftDown || message == windowsMessageLeftUp {
		button = 1
	}
	if message == windowsMessageRightDown || message == windowsMessageRightUp {
		button = 2
	}
	if message == windowsMessageMiddleDown || message == windowsMessageMiddleUp {
		button = 3
	}
	if message == windowsMessageMouseMove && !w.tracking {
		tracking := windowsTrackMouseEvent{Size: windowsTrackMouseEventSize, Flags: windowsTrackLeave, Window: w.native}
		if windowsBeginTrackMouseEvent(&tracking) != 0 {
			w.tracking = true
		}
	}
	w.pointerInside = true
	w.queue(Event{Type: eventType, X: Scalar(x), Y: Scalar(y), Button: button, Modifiers: modifiers})
}

func (w *Window) dispatchWindowsMessage(message *windowsMessage) bool {
	typ := int(message.Message)
	modifiers := windowsModifiers()
	if typ == windowsMessageClose {
		w.queue(Event{Type: EventWindowClose})
		return false
	}
	if typ == windowsMessageDestroy {
		if !w.closed {
			w.closed = true
			w.queue(Event{Type: EventWindowClose})
		}
		return true
	}
	if typ == windowsMessageSize {
		width := message.LParam & 0xffff
		height := message.LParam >> 16 & 0xffff
		w.resizeSurface(width, height)
	}
	if typ == windowsMessageSetFocus {
		w.focused = true
		w.queue(Event{Type: EventWindowFocusGained})
	}
	if typ == windowsMessageKillFocus {
		w.focused = false
		w.queue(Event{Type: EventWindowFocusLost})
	}
	if typ == windowsMessagePaint {
		dirty := windowsRect{}
		if windowsGetUpdateRect(w.native, &dirty, 0) != 0 {
			w.queue(Event{Type: EventWindowExpose, Dirty: R(Scalar(dirty.Left), Scalar(dirty.Top), Scalar(dirty.Right-dirty.Left), Scalar(dirty.Bottom-dirty.Top))})
		}
		windowsValidateRect(w.native, nil)
		return false
	}
	if typ == windowsMessageSetCursor {
		nativeCursor := windowsLoadCursor(0, windowsCursorResource(w.cursor))
		if nativeCursor != 0 {
			windowsSetCursor(nativeCursor)
			return false
		}
	}
	if typ == windowsMessageKeyDown || typ == windowsMessageSystemKeyDown || typ == windowsMessageKeyUp || typ == windowsMessageSystemKeyUp {
		eventType := EventKeyDown
		if typ == windowsMessageKeyUp || typ == windowsMessageSystemKeyUp {
			eventType = EventKeyUp
		}
		w.queue(Event{Type: eventType, Key: windowsKeyFromVirtual(message.WParam), Modifiers: modifiers, Repeat: message.LParam&(1<<30) != 0})
	}
	if typ == windowsMessageCharacter {
		w.queueWindowsCharacter(message.WParam&0xffff, modifiers)
	}
	if typ == windowsMessageMouseMove || typ == windowsMessageLeftDown || typ == windowsMessageLeftUp || typ == windowsMessageRightDown || typ == windowsMessageRightUp || typ == windowsMessageMiddleDown || typ == windowsMessageMiddleUp {
		w.queuePointer(typ, windowsSignedWord(message.LParam), windowsSignedWord(message.LParam>>16), modifiers)
	}
	if typ == windowsMessageMouseLeave {
		w.tracking = false
		w.pointerInside = false
		w.queue(Event{Type: EventPointerLeave, Modifiers: modifiers})
	}
	if typ == windowsMessageMouseWheel || typ == windowsMessageMouseHWheel {
		point := windowsPoint{X: int32(windowsSignedWord(message.LParam)), Y: int32(windowsSignedWord(message.LParam >> 16))}
		windowsScreenToClient(w.native, &point)
		delta := wheelDeltaPixels(Scalar(windowsSignedWord(message.WParam>>16))/120.0, false)
		wheelX := Scalar(0)
		wheelY := Scalar(0)
		if typ == windowsMessageMouseHWheel {
			wheelX = -delta
		} else {
			wheelY = delta
		}
		w.queue(Event{Type: EventPointerWheel, X: Scalar(point.X), Y: Scalar(point.Y), WheelX: wheelX, WheelY: wheelY, Modifiers: modifiers})
	}
	if typ == windowsMessageTimer {
		w.CancelTimer(message.WParam)
		w.queue(Event{Type: EventTimer, TimerID: message.WParam})
	}
	return true
}

func pumpWindowsMessage(message *windowsMessage) {
	if int(message.Message) == windowsMessageQuit {
		for i := 0; i < len(windowsWindows); i++ {
			if !windowsWindows[i].closed {
				windowsWindows[i].queue(Event{Type: EventWindowClose})
			}
		}
		return
	}
	target := windowsWindowForNative(message.Window)
	dispatch := true
	if target != nil {
		dispatch = target.dispatchWindowsMessage(message)
	}
	if dispatch {
		windowsTranslateMessage(message)
		windowsDispatchMessage(message)
	}
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
	message := windowsMessage{}
	for windowsPeekMessage(&message, 0, 0, 0, windowsPeekRemove) != 0 {
		pumpWindowsMessage(&message)
		if event, ok := w.nextQueuedEvent(); ok {
			return event, true
		}
	}
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
		message := windowsMessage{}
		result := windowsGetMessage(&message, 0, 0, 0)
		if result <= 0 {
			return Event{}, false
		}
		pumpWindowsMessage(&message)
	}
	return w.nextQueuedEvent()
}

func SetClipboardText(text string) bool {
	data := windowsClipboardEncode(text)
	memory := windowsGlobalAlloc(windowsGlobalMoveable, len(data))
	if memory == 0 {
		return false
	}
	pointer := windowsGlobalLock(memory)
	if pointer == 0 {
		windowsGlobalFree(memory)
		return false
	}
	windowsMoveMemory(pointer, &data[0], len(data))
	windowsGlobalUnlock(memory)
	if windowsOpenClipboard(0) == 0 {
		windowsGlobalFree(memory)
		return false
	}
	windowsEmptyClipboard()
	result := windowsSetClipboardData(windowsClipboardTextFormat, memory)
	windowsCloseClipboard()
	if result == 0 {
		windowsGlobalFree(memory)
		return false
	}
	return true
}

func ClipboardText() (string, bool) {
	if windowsOpenClipboard(0) == 0 {
		return "", false
	}
	memory := windowsGetClipboardData(windowsClipboardTextFormat)
	if memory == 0 {
		windowsCloseClipboard()
		return "", false
	}
	pointer := windowsGlobalLock(memory)
	size := windowsGlobalSize(memory)
	if pointer == 0 || size <= 0 {
		if pointer != 0 {
			windowsGlobalUnlock(memory)
		}
		windowsCloseClipboard()
		return "", false
	}
	data := make([]byte, size)
	windowsReadMemory(&data[0], pointer, size)
	windowsGlobalUnlock(memory)
	windowsCloseClipboard()
	return windowsClipboardDecode(data), true
}

func (w *Window) Present() bool {
	if w == nil || w.closed || w.context == 0 || w.device == 0 {
		return false
	}
	if !w.surface.dirtyValid {
		return true
	}
	row := w.surface.Stride
	for regionIndex := 0; regionIndex < len(w.surface.dirtyRects); regionIndex++ {
		dirty := intersectPixelRect(pixelRect{maxX: w.width, maxY: w.height}, w.surface.dirtyRects[regionIndex])
		for y := dirty.minY; y < dirty.maxY; y++ {
			source := y * row
			destination := (w.height - y - 1) * row
			start := dirty.minX * 4
			end := dirty.maxX * 4
			copy(w.bottomUp[destination+start:destination+end], w.surface.Pixels[source+start:source+end])
		}
	}
	if windowsWGLMakeCurrent(w.device, w.context) == 0 {
		return false
	}
	glMatrixMode(glProjection)
	glLoadIdentity()
	glMatrixMode(glModelView)
	glLoadIdentity()
	// Update only the damaged pixels in the back buffer. After the swap, apply
	// the same small update to the new back buffer so both buffers remain in
	// sync and future partial frames never expose stale pixels.
	glPixelStorei(glUnpackAlignment, 1)
	glPixelStorei(glUnpackRowLength, w.width)
	glDrawBuffer(glBack)
	for regionIndex := 0; regionIndex < len(w.surface.dirtyRects); regionIndex++ {
		dirty := intersectPixelRect(pixelRect{maxX: w.width, maxY: w.height}, w.surface.dirtyRects[regionIndex])
		if dirty.maxX <= dirty.minX || dirty.maxY <= dirty.minY {
			continue
		}
		glViewport(dirty.minX, w.height-dirty.maxY, dirty.maxX-dirty.minX, dirty.maxY-dirty.minY)
		glRasterPos2i(-1, -1)
		start := (w.height-dirty.maxY)*row + dirty.minX*4
		glDrawPixels(dirty.maxX-dirty.minX, dirty.maxY-dirty.minY, glRGBA, glUnsignedByte, w.bottomUp[start:])
	}
	if windowsSwapBuffers(w.device) == 0 {
		glPixelStorei(glUnpackRowLength, 0)
		return false
	}
	glDrawBuffer(glBack)
	for regionIndex := 0; regionIndex < len(w.surface.dirtyRects); regionIndex++ {
		dirty := intersectPixelRect(pixelRect{maxX: w.width, maxY: w.height}, w.surface.dirtyRects[regionIndex])
		if dirty.maxX <= dirty.minX || dirty.maxY <= dirty.minY {
			continue
		}
		glViewport(dirty.minX, w.height-dirty.maxY, dirty.maxX-dirty.minX, dirty.maxY-dirty.minY)
		glRasterPos2i(-1, -1)
		start := (w.height-dirty.maxY)*row + dirty.minX*4
		glDrawPixels(dirty.maxX-dirty.minX, dirty.maxY-dirty.minY, glRGBA, glUnsignedByte, w.bottomUp[start:])
	}
	glPixelStorei(glUnpackRowLength, 0)
	w.surface.ResetDirty()
	return true
}

func (w *Window) ReadPixels() *Image {
	if w == nil || w.closed || w.context == 0 || w.device == 0 || w.width <= 0 || w.height <= 0 {
		return nil
	}
	if windowsWGLMakeCurrent(w.device, w.context) == 0 {
		return nil
	}
	bottomUp := make([]byte, w.width*w.height*4)
	glViewport(0, 0, w.width, w.height)
	glMatrixMode(glProjection)
	glLoadIdentity()
	glMatrixMode(glModelView)
	glLoadIdentity()
	glDrawBuffer(glBack)
	glRasterPos2i(-1, -1)
	glPixelStorei(glUnpackAlignment, 1)
	glDrawPixels(w.width, w.height, glRGBA, glUnsignedByte, w.bottomUp)
	glFinish()
	// Hidden windows do not have a reliably readable front buffer. Rebuild the
	// current software frame in the back buffer so capture is deterministic.
	glReadBuffer(glBack)
	glPixelStorei(glPackAlignment, 1)
	glReadPixels(0, 0, w.width, w.height, glRGBA, glUnsignedByte, bottomUp)
	image := NewImage(w.width, w.height, nil)
	row := w.width * 4
	for y := 0; y < w.height; y++ {
		source := (w.height - y - 1) * row
		destination := y * row
		copy(image.Pixels[destination:destination+row], bottomUp[source:source+row])
	}
	return image
}

func (w *Window) Close() {
	if w == nil || !w.active {
		return
	}
	w.closed = true
	w.shown = false
	for i := 0; i < len(w.timerActive); i++ {
		if w.timerActive[i] {
			windowsKillTimer(w.native, w.timerIDs[i])
			w.timerActive[i] = false
		}
	}
	if w.context != 0 {
		windowsWGLMakeCurrent(0, 0)
		windowsWGLDeleteContext(w.context)
		w.context = 0
	}
	if w.device != 0 && w.native != 0 {
		windowsReleaseDC(w.native, w.device)
		w.device = 0
	}
	if w.native != 0 {
		windowsDestroyWindow(w.native)
		w.native = 0
	}
	w.active = false
	forgetWindowsWindow(w)
}
