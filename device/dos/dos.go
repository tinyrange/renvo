// Package dos exposes MS-DOS and PC BIOS services that do not have a portable
// standard-library equivalent. Portable file operations should continue to use
// os; this package provides DOS-specific metadata, enumeration, and devices.
package dos

import "unsafe"

const (
	FlagCarry = 1 << 0
	FlagZero  = 1 << 6
)

// Registers is the 8086 register block used by Interrupt.
type Registers struct {
	AX, BX, CX, DX uint16
	SI, DI         uint16
	Flags          uint16
	ES             uint16
}

// Error is an MS-DOS error code returned in AX when carry is set.
type Error uint16

func (e Error) Error() string {
	switch e {
	case 2:
		return "file not found"
	case 3:
		return "path not found"
	case 4:
		return "too many open files"
	case 5:
		return "access denied"
	case 6:
		return "invalid handle"
	case 8:
		return "insufficient memory"
	case 15:
		return "invalid drive"
	case 18:
		return "no more files"
	}
	return "MS-DOS error"
}

func result(regs *Registers) error {
	if regs.Flags&FlagCarry != 0 {
		return Error(regs.AX)
	}
	return nil
}

func nearPointer(data []byte) uint16 {
	if len(data) == 0 {
		return 0
	}
	return uint16(uintptr(unsafe.Pointer(&data[0])))
}

func cString(value string) []byte {
	out := make([]byte, len(value)+1)
	copy(out, []byte(value))
	return out
}

// Interrupt invokes one of the supported PC BIOS or DOS interrupts.
func Interrupt(vector byte, regs *Registers) {
	if regs == nil {
		return
	}
	switch vector {
	case 0x10:
		interrupt10(regs)
	case 0x13:
		interrupt13(regs)
	case 0x14:
		interrupt14(regs)
	case 0x15:
		interrupt15(regs)
	case 0x16:
		interrupt16(regs)
	case 0x1a:
		interrupt1A(regs)
	case 0x21:
		interrupt21(regs)
	case 0x17:
		interrupt17(regs)
	case 0x33:
		interrupt33(regs)
	}
}

func Version() (major, minor byte) {
	regs := Registers{AX: 0x3000}
	interrupt21(&regs)
	return byte(regs.AX), byte(regs.AX >> 8)
}

// AllocateParagraphs allocates conventional memory in 16-byte paragraphs.
func AllocateParagraphs(paragraphs uint16) (uint16, error) {
	regs := Registers{AX: 0x4800, BX: paragraphs}
	interrupt21(&regs)
	return regs.AX, result(&regs)
}

func FreeParagraphs(segment uint16) error {
	regs := Registers{AX: 0x4900, ES: segment}
	interrupt21ES(&regs)
	return result(&regs)
}

func ResizeParagraphs(segment, paragraphs uint16) error {
	regs := Registers{AX: 0x4a00, BX: paragraphs, ES: segment}
	interrupt21ES(&regs)
	return result(&regs)
}

// BIOSTicks returns the tick count since midnight and the midnight rollover
// counter reported by INT 1Ah.
func BIOSTicks() (uint32, byte) {
	regs := Registers{}
	interrupt1A(&regs)
	return uint32(regs.CX)<<16 | uint32(regs.DX), byte(regs.AX)
}

const (
	AttributeReadOnly  = 0x01
	AttributeHidden    = 0x02
	AttributeSystem    = 0x04
	AttributeVolume    = 0x08
	AttributeDirectory = 0x10
	AttributeArchive   = 0x20
)

func pathCall(function uint16, path string) error {
	name := cString(path)
	regs := Registers{AX: function << 8, DX: nearPointer(name)}
	interrupt21(&regs)
	return result(&regs)
}

func MakeDir(path string) error   { return pathCall(0x39, path) }
func ChangeDir(path string) error { return pathCall(0x3b, path) }
func RemoveDir(path string) error { return pathCall(0x3a, path) }
func Remove(path string) error    { return pathCall(0x41, path) }
func SetDrive(drive byte) byte {
	regs := Registers{AX: 0x0e00, DX: uint16(drive)}
	interrupt21(&regs)
	return byte(regs.AX)
}
func Drive() byte {
	regs := Registers{AX: 0x1900}
	interrupt21(&regs)
	return byte(regs.AX)
}
func CurrentDirectory(drive byte) (string, error) {
	buffer := make([]byte, 64)
	regs := Registers{AX: 0x4700, DX: uint16(drive), SI: nearPointer(buffer)}
	interrupt21(&regs)
	if err := result(&regs); err != nil {
		return "", err
	}
	end := 0
	for end < len(buffer) && buffer[end] != 0 {
		end++
	}
	return string(buffer[:end]), nil
}
func Rename(oldPath, newPath string) error {
	oldName, newName := cString(oldPath), cString(newPath)
	regs := Registers{AX: 0x5600, DX: nearPointer(oldName), DI: nearPointer(newName)}
	interrupt21(&regs)
	return result(&regs)
}
func FileAttributes(path string) (uint16, error) {
	name := cString(path)
	regs := Registers{AX: 0x4300, DX: nearPointer(name)}
	interrupt21(&regs)
	return regs.CX, result(&regs)
}
func SetFileAttributes(path string, attributes uint16) error {
	name := cString(path)
	regs := Registers{AX: 0x4301, CX: attributes, DX: nearPointer(name)}
	interrupt21(&regs)
	return result(&regs)
}

// DirEntry is one DOS FindFirst/FindNext result.
type DirEntry struct {
	Name       string
	Attributes uint16
	Size       int64
	Date       uint16
	Time       uint16
}

func (e DirEntry) IsDir() bool { return e.Attributes&AttributeDirectory != 0 }

type Finder struct {
	dta     []byte
	pattern []byte
	attrs   uint16
	started bool
	done    bool
}

func Find(pattern string, attributes uint16) *Finder {
	return &Finder{dta: make([]byte, 43), pattern: cString(pattern), attrs: attributes}
}

func (f *Finder) Next() (DirEntry, bool, error) {
	if f == nil || f.done {
		return DirEntry{}, false, nil
	}
	set := Registers{AX: 0x1a00, DX: nearPointer(f.dta)}
	interrupt21(&set)
	if err := result(&set); err != nil {
		return DirEntry{}, false, err
	}
	regs := Registers{AX: 0x4f00}
	if !f.started {
		regs.AX = 0x4e00
		regs.CX = f.attrs
		regs.DX = nearPointer(f.pattern)
		f.started = true
	}
	interrupt21(&regs)
	if err := result(&regs); err != nil {
		if code, ok := err.(Error); ok && code == 18 {
			f.done = true
			return DirEntry{}, false, nil
		}
		return DirEntry{}, false, err
	}
	return parseDirEntry(f.dta), true, nil
}

func parseDirEntry(dta []byte) DirEntry {
	end := 30
	for end < len(dta) && dta[end] != 0 {
		end++
	}
	entry := DirEntry{Name: string(dta[30:end]), Attributes: uint16(dta[21]),
		Time: uint16(dta[22]) | uint16(dta[23])<<8,
		Date: uint16(dta[24]) | uint16(dta[25])<<8}
	entry.Size = int64(uint32(dta[26]) | uint32(dta[27])<<8 | uint32(dta[28])<<16 | uint32(dta[29])<<24)
	return entry
}

type DateTime struct {
	Year, Month, Day, Weekday int
	Hour, Minute, Second      int
	Hundredth                 int
}

func Now() DateTime {
	date := Registers{AX: 0x2a00}
	interrupt21(&date)
	time := Registers{AX: 0x2c00}
	interrupt21(&time)
	return DateTime{Year: int(date.CX), Month: int(date.DX >> 8), Day: int(date.DX & 255),
		Weekday: int(date.AX & 255), Hour: int(time.CX >> 8), Minute: int(time.CX & 255),
		Second: int(time.DX >> 8), Hundredth: int(time.DX & 255)}
}

func WriteConsole(text string) error {
	data := []byte(text)
	regs := Registers{AX: 0x4000, BX: 1, CX: uint16(len(data)), DX: nearPointer(data)}
	interrupt21(&regs)
	return result(&regs)
}
