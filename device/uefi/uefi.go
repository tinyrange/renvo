// Package uefi exposes the firmware tables and common protocols available to
// UEFI applications. It intentionally stays close to the UEFI ABI: callers can
// use the small typed helpers or pass protocol function pointers to Call.
package uefi

import "unsafe"

type Handle uintptr
type Event uintptr
type Status uintptr

const (
	Success             Status = 0
	LoadError           Status = Status(1) | Status(1)<<(uintptrBits-1)
	InvalidParameter    Status = Status(2) | Status(1)<<(uintptrBits-1)
	Unsupported         Status = Status(3) | Status(1)<<(uintptrBits-1)
	BadBufferSize       Status = Status(4) | Status(1)<<(uintptrBits-1)
	BufferTooSmall      Status = Status(5) | Status(1)<<(uintptrBits-1)
	NotReady            Status = Status(6) | Status(1)<<(uintptrBits-1)
	DeviceError         Status = Status(7) | Status(1)<<(uintptrBits-1)
	WriteProtected      Status = Status(8) | Status(1)<<(uintptrBits-1)
	OutOfResources      Status = Status(9) | Status(1)<<(uintptrBits-1)
	VolumeCorrupted     Status = Status(10) | Status(1)<<(uintptrBits-1)
	VolumeFull          Status = Status(11) | Status(1)<<(uintptrBits-1)
	NoMedia             Status = Status(12) | Status(1)<<(uintptrBits-1)
	MediaChanged        Status = Status(13) | Status(1)<<(uintptrBits-1)
	NotFound            Status = Status(14) | Status(1)<<(uintptrBits-1)
	AccessDenied        Status = Status(15) | Status(1)<<(uintptrBits-1)
	NoResponse          Status = Status(16) | Status(1)<<(uintptrBits-1)
	NoMapping           Status = Status(17) | Status(1)<<(uintptrBits-1)
	Timeout             Status = Status(18) | Status(1)<<(uintptrBits-1)
	NotStarted          Status = Status(19) | Status(1)<<(uintptrBits-1)
	AlreadyStarted      Status = Status(20) | Status(1)<<(uintptrBits-1)
	Aborted             Status = Status(21) | Status(1)<<(uintptrBits-1)
	ICMPError           Status = Status(22) | Status(1)<<(uintptrBits-1)
	TFTPError           Status = Status(23) | Status(1)<<(uintptrBits-1)
	ProtocolError       Status = Status(24) | Status(1)<<(uintptrBits-1)
	IncompatibleVersion Status = Status(25) | Status(1)<<(uintptrBits-1)
	SecurityViolation   Status = Status(26) | Status(1)<<(uintptrBits-1)
	CRCError            Status = Status(27) | Status(1)<<(uintptrBits-1)
	EndOfMedia          Status = Status(28) | Status(1)<<(uintptrBits-1)
	EndOfFile           Status = Status(31) | Status(1)<<(uintptrBits-1)
	CompromisedData     Status = Status(33) | Status(1)<<(uintptrBits-1)
	IPAddressConflict   Status = Status(34) | Status(1)<<(uintptrBits-1)
	HTTPError           Status = Status(35) | Status(1)<<(uintptrBits-1)
)

const uintptrBits = 32 << (^uintptr(0) >> 63)

func (s Status) Error() string {
	switch s {
	case Success:
		return "success"
	case LoadError:
		return "load error"
	case InvalidParameter:
		return "invalid parameter"
	case Unsupported:
		return "unsupported"
	case BadBufferSize:
		return "bad buffer size"
	case BufferTooSmall:
		return "buffer too small"
	case NotReady:
		return "not ready"
	case DeviceError:
		return "device error"
	case WriteProtected:
		return "write protected"
	case OutOfResources:
		return "out of resources"
	case VolumeCorrupted:
		return "volume corrupted"
	case VolumeFull:
		return "volume full"
	case NoMedia:
		return "no media"
	case MediaChanged:
		return "media changed"
	case NoResponse:
		return "no response"
	case NoMapping:
		return "no mapping"
	case NotFound:
		return "not found"
	case AccessDenied:
		return "access denied"
	case Timeout:
		return "timeout"
	case NotStarted:
		return "not started"
	case AlreadyStarted:
		return "already started"
	case Aborted:
		return "aborted"
	case ICMPError:
		return "ICMP error"
	case TFTPError:
		return "TFTP error"
	case ProtocolError:
		return "protocol error"
	case IncompatibleVersion:
		return "incompatible version"
	case SecurityViolation:
		return "security violation"
	case CRCError:
		return "CRC error"
	case EndOfMedia:
		return "end of media"
	case EndOfFile:
		return "end of file"
	case CompromisedData:
		return "compromised data"
	case IPAddressConflict:
		return "IP address conflict"
	case HTTPError:
		return "HTTP error"
	}
	kind := "UEFI warning "
	value := uintptr(s)
	if s.Failed() {
		kind = "UEFI error "
		value &^= uintptr(1) << (uintptrBits - 1)
	}
	return kind + decimalStatus(value)
}

func (s Status) Failed() bool { return uintptr(s)>>(uintptrBits-1) != 0 }

func decimalStatus(value uintptr) string {
	if value == 0 {
		return "0"
	}
	var digits [24]byte
	at := len(digits)
	for value != 0 {
		at--
		digits[at] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[at:])
}

// GUID stores the exact 16-byte in-memory representation used by UEFI.
type GUID struct{ Low, High uint64 }

// SystemTable is an opaque firmware address. Accessors use UEFI's specified
// byte offsets because Renvo intentionally gives small Go struct fields
// word-sized slots, which is not the packed C layout used by firmware tables.
type SystemTable uintptr

func ImageHandle() Handle { return Handle(imageHandle()) }

func CurrentSystemTable() SystemTable { return SystemTable(systemTable()) }
func (s SystemTable) Valid() bool     { return s != 0 }
func (s SystemTable) ConsoleInput() *SimpleTextInputProtocol {
	return (*SimpleTextInputProtocol)(unsafe.Pointer(loadWord(uintptr(s), 48)))
}
func (s SystemTable) ConsoleOutput() *SimpleTextOutputProtocol {
	return (*SimpleTextOutputProtocol)(unsafe.Pointer(loadWord(uintptr(s), 64)))
}
func (s SystemTable) ErrorOutput() *SimpleTextOutputProtocol {
	return (*SimpleTextOutputProtocol)(unsafe.Pointer(loadWord(uintptr(s), 80)))
}
func (s SystemTable) Runtime() *RuntimeServices {
	return (*RuntimeServices)(unsafe.Pointer(loadWord(uintptr(s), 88)))
}
func (s SystemTable) Boot() *BootServices {
	return (*BootServices)(unsafe.Pointer(loadWord(uintptr(s), 96)))
}

// ConfigurationTable returns the vendor table registered for guid, or zero
// when the firmware did not publish one.
func (s SystemTable) ConfigurationTable(guid GUID) uintptr {
	if !s.Valid() {
		return 0
	}
	count := loadWord(uintptr(s), 104)
	entries := loadWord(uintptr(s), 112)
	if entries == 0 || count > 4096 {
		return 0
	}
	for i := uintptr(0); i < count; i++ {
		entry := entries + i*24
		if load64(entry, 0) == guid.Low && load64(entry, 8) == guid.High {
			return loadWord(entry, 16)
		}
	}
	return 0
}

func FirmwareVendor() string {
	table := CurrentSystemTable()
	if !table.Valid() {
		return ""
	}
	return string16((*uint16)(unsafe.Pointer(loadWord(uintptr(table), 24))))
}

func pointer(value unsafe.Pointer) uintptr { return uintptr(value) }
func loadWord(base, offset uintptr) uintptr {
	if base == 0 {
		return 0
	}
	return *(*uintptr)(unsafe.Pointer(base + offset))
}
func load32(base, offset uintptr) uint32 {
	if base == 0 {
		return 0
	}
	return *(*uint32)(unsafe.Pointer(base + offset))
}

func load64(base, offset uintptr) uint64 {
	if base == 0 {
		return 0
	}
	return *(*uint64)(unsafe.Pointer(base + offset))
}
func store8(base, offset uintptr, value byte) {
	*(*byte)(unsafe.Pointer(base + offset)) = value
}
func store32(base, offset uintptr, value uint32) {
	*(*uint32)(unsafe.Pointer(base + offset)) = value
}
func store64(base, offset uintptr, value uint64) {
	*(*uint64)(unsafe.Pointer(base + offset)) = value
}

func string16(value *uint16) string {
	if value == nil {
		return ""
	}
	units := (*[1 << 20]uint16)(unsafe.Pointer(value))
	out := make([]byte, 0, 32)
	for i := 0; units[i] != 0; i++ {
		u := units[i]
		if u >= 0xd800 && u <= 0xdbff && units[i+1] >= 0xdc00 && units[i+1] <= 0xdfff {
			r := rune(0x10000 + (uint32(u-0xd800) << 10) + uint32(units[i+1]-0xdc00))
			out = appendUTF8(out, r)
			i++
		} else {
			out = appendUTF8(out, rune(u))
		}
	}
	return string(out)
}

func appendUTF8(out []byte, r rune) []byte {
	if r <= 0x7f {
		return append(out, byte(r))
	}
	if r <= 0x7ff {
		return append(out, byte(0xc0|(r>>6)), byte(0x80|(r&0x3f)))
	}
	if r <= 0xffff {
		return append(out, byte(0xe0|(r>>12)), byte(0x80|((r>>6)&0x3f)), byte(0x80|(r&0x3f)))
	}
	return append(out, byte(0xf0|(r>>18)), byte(0x80|((r>>12)&0x3f)),
		byte(0x80|((r>>6)&0x3f)), byte(0x80|(r&0x3f)))
}

func utf16z(value string) []uint16 {
	out := make([]uint16, len(value)+1)
	return out[:encodeUTF16Z(out, value)]
}

// encodeUTF16Z writes the NUL-terminated UTF-16 representation of value into
// out and returns the number of code units, including the terminator. A zero
// result means the destination was too small.
func encodeUTF16Z(out []uint16, value string) int {
	to := 0
	for at := 0; at < len(value); {
		first := value[at]
		at++
		r := rune(first)
		if first >= 0xc2 && first <= 0xdf && at < len(value) {
			r = rune(first&0x1f)<<6 | rune(value[at]&0x3f)
			at++
		} else if first >= 0xe0 && first <= 0xef && at+1 < len(value) {
			r = rune(first&0x0f)<<12 | rune(value[at]&0x3f)<<6 | rune(value[at+1]&0x3f)
			at += 2
		} else if first >= 0xf0 && first <= 0xf4 && at+2 < len(value) {
			r = rune(first&7)<<18 | rune(value[at]&0x3f)<<12 |
				rune(value[at+1]&0x3f)<<6 | rune(value[at+2]&0x3f)
			at += 3
		} else if first >= 0x80 {
			r = 0xfffd
		}
		if r <= 0xffff {
			if to+1 >= len(out) {
				return 0
			}
			out[to] = uint16(r)
			to++
		} else {
			if to+2 >= len(out) {
				return 0
			}
			r -= 0x10000
			out[to] = uint16(0xd800 + (r >> 10))
			out[to+1] = uint16(0xdc00 + (r & 0x3ff))
			to += 2
		}
	}
	if to >= len(out) {
		return 0
	}
	out[to] = 0
	return to + 1
}

// Call invokes a firmware function pointer using the UEFI x64 ABI.
func Call(function uintptr, arguments ...uintptr) Status {
	switch len(arguments) {
	case 0:
		return Status(call0(function))
	case 1:
		return Status(call1(function, arguments[0]))
	case 2:
		return Status(call2(function, arguments[0], arguments[1]))
	case 3:
		return Status(call3(function, arguments[0], arguments[1], arguments[2]))
	case 4:
		return Status(call4(function, arguments[0], arguments[1], arguments[2], arguments[3]))
	case 5:
		return Status(call5(function, arguments[0], arguments[1], arguments[2], arguments[3], arguments[4]))
	}
	return InvalidParameter
}
