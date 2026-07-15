//go:build rtg && windows && amd64

package graphics

const windowsDefWindowProcName = "DefWindowProcW"
const windowsClipboardTextFormat = 13

// rtg:linkstatic kernel32.dll,GetModuleHandleW
func windowsGetModuleHandle(name *byte) int { return 0 }

// rtg:linkstatic user32.dll,RegisterClassW
func windowsRegisterClassRaw(value *byte) int { return 0 }

// rtg:linkstatic user32.dll,CreateWindowExW
func windowsCreateWindow(exStyle, className, title, style, x, y, width, height, parent, menu, instance, param int) int {
	return 0
}

// rtg:linkstatic user32.dll,DefWindowProcW
func windowsDefWindowProc(window, message, wParam, lParam int) int { return 0 }

// rtg:linkstatic user32.dll,SetWindowTextW
func windowsSetWindowText(window int, text *byte) int { return 0 }

// rtg:linkstatic user32.dll,PeekMessageW
func windowsPeekMessageRaw(message *byte, window, first, last, remove int) int { return 0 }

// rtg:linkstatic user32.dll,GetMessageW
func windowsGetMessageRaw(message *byte, window, first, last int) int { return 0 }

// rtg:linkstatic user32.dll,DispatchMessageW
func windowsDispatchMessageRaw(message *byte) int { return 0 }

// rtg:linkstatic user32.dll,LoadCursorW
func windowsLoadCursor(instance, resource int) int { return 0 }

func windowsNativeString(text string) []byte {
	return windowsUTF16(text)
}

func windowsClipboardEncode(text string) []byte {
	return windowsUTF16(text)
}

func windowsClipboardDecode(data []byte) string {
	return windowsUTF16BytesToString(data)
}

func (w *Window) queueWindowsCharacter(unit int, modifiers Modifiers) {
	w.queueUTF16(unit, modifiers)
}
