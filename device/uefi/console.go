package uefi

import "unsafe"

type InputKey struct {
	ScanCode    uint16
	UnicodeChar uint16
}

type SimpleTextInputProtocol struct {
	Reset         uintptr
	ReadKeyStroke uintptr
	WaitForKey    Event
}

type SimpleTextOutputProtocol struct {
	Reset             uintptr
	OutputString      uintptr
	TestString        uintptr
	QueryMode         uintptr
	SetMode           uintptr
	SetAttribute      uintptr
	ClearScreen       uintptr
	SetCursorPosition uintptr
	EnableCursor      uintptr
	Mode              uintptr
}

func WriteString(value string) Status {
	text := utf16z(value)
	return WriteUTF16(text)
}

// WriteUTF16 writes a NUL-terminated UTF-16 buffer through the console.
func WriteUTF16(text []uint16) Status {
	table := CurrentSystemTable()
	console := table.ConsoleOutput()
	if console == nil || len(text) == 0 {
		return NotReady
	}
	return Status(call2(console.OutputString,
		pointer(unsafe.Pointer(console)), pointer(unsafe.Pointer(&text[0]))))
}

func ClearScreen() Status {
	table := CurrentSystemTable()
	console := table.ConsoleOutput()
	if console == nil {
		return NotReady
	}
	return Status(call1(console.ClearScreen, pointer(unsafe.Pointer(console))))
}

func ReadKey() (InputKey, Status) {
	var key InputKey
	table := CurrentSystemTable()
	console := table.ConsoleInput()
	if console == nil {
		return key, NotReady
	}
	status := Status(call2(console.ReadKeyStroke,
		pointer(unsafe.Pointer(console)), pointer(unsafe.Pointer(&key))))
	return key, status
}
