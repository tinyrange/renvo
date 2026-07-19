//go:build renvo && windows && 386

package graphics

const windowsDefWindowProcName = "DefWindowProcA"
const windowsClipboardTextFormat = 1

// renvo:linkstatic kernel32.dll,GetModuleHandleA
func windowsGetModuleHandle(name *byte) int { return 0 }

// renvo:linkstatic kernel32.dll,WideCharToMultiByte
func windowsWideCharToMultiByte(codePage, flags int, wide *byte, wideLength int, bytes *byte, byteLength int, defaultByte, usedDefault *byte) int {
	return 0
}

// renvo:linkstatic kernel32.dll,MultiByteToWideChar
func windowsMultiByteToWideChar(codePage, flags int, bytes *byte, byteLength int, wide *byte, wideLength int) int {
	return 0
}

// renvo:linkstatic user32.dll,RegisterClassA
func windowsRegisterClassRaw(value *byte) int { return 0 }

// renvo:linkstatic user32.dll,CreateWindowExA
func windowsCreateWindow(exStyle, className, title, style, x, y, width, height, parent, menu, instance, param int) int {
	return 0
}

// renvo:linkstatic user32.dll,DefWindowProcA
func windowsDefWindowProc(window, message, wParam, lParam int) int { return 0 }

// renvo:linkstatic user32.dll,SetWindowTextA
func windowsSetWindowText(window int, text *byte) int { return 0 }

// renvo:linkstatic user32.dll,PeekMessageA
func windowsPeekMessageRaw(message *byte, window, first, last, remove int) int { return 0 }

// renvo:linkstatic user32.dll,GetMessageA
func windowsGetMessageRaw(message *byte, window, first, last int) int { return 0 }

// renvo:linkstatic user32.dll,DispatchMessageA
func windowsDispatchMessageRaw(message *byte) int { return 0 }

// renvo:linkstatic user32.dll,LoadCursorA
func windowsLoadCursor(instance, resource int) int { return 0 }

func windowsNativeString(text string) []byte {
	wide := windowsUTF16(text)
	required := windowsWideCharToMultiByte(0, 0, &wide[0], -1, nil, 0, nil, nil)
	if required <= 0 {
		return windowsASCII(text)
	}
	out := make([]byte, required)
	if windowsWideCharToMultiByte(0, 0, &wide[0], -1, &out[0], len(out), nil, nil) <= 0 {
		return windowsASCII(text)
	}
	return out
}

func windowsNativeBytesToString(data []byte, length int) (string, bool) {
	if length <= 0 || len(data) == 0 {
		return "", true
	}
	const rejectInvalidCharacters = 8
	required := windowsMultiByteToWideChar(0, rejectInvalidCharacters, &data[0], length, nil, 0)
	if required <= 0 {
		return "", false
	}
	wide := make([]byte, required*2+2)
	if windowsMultiByteToWideChar(0, rejectInvalidCharacters, &data[0], length, &wide[0], required) != required {
		return "", false
	}
	return windowsUTF16BytesToString(wide), true
}

func windowsClipboardEncode(text string) []byte {
	return windowsNativeString(text)
}

func windowsClipboardDecode(data []byte) string {
	length := 0
	for length < len(data) && data[length] != 0 {
		length++
	}
	text, _ := windowsNativeBytesToString(data, length)
	return text
}

func (w *Window) queueWindowsCharacter(unit int, modifiers Modifiers) {
	current := byte(unit)
	if w.pendingUTF16 != 0 {
		data := []byte{byte(w.pendingUTF16), current}
		w.pendingUTF16 = 0
		if text, ok := windowsNativeBytesToString(data, len(data)); ok {
			w.queue(Event{Type: EventTextInput, Text: text, Modifiers: modifiers})
			return
		}
	}
	data := []byte{current}
	if text, ok := windowsNativeBytesToString(data, len(data)); ok {
		w.queue(Event{Type: EventTextInput, Text: text, Modifiers: modifiers})
		return
	}
	w.pendingUTF16 = int(current)
}
