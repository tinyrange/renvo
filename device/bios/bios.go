// Package bios exposes legacy IBM-PC firmware services to bios/8086 programs.
// It uses real-mode interrupts and segmented addresses directly; it does not
// assume DOS, a filesystem, or a protected-mode host environment.
package bios

import "unsafe"

const (
	FlagCarry = 1 << 0
	FlagZero  = 1 << 6
)

// Registers is the real-mode register block used by Interrupt.
type Registers struct {
	AX, BX, CX, DX uint16
	SI, DI         uint16
	Flags          uint16
	ES             uint16
}

// Error is a BIOS status byte returned in AH when carry is set.
type Error byte

func (e Error) Error() string {
	switch e {
	case 0x01:
		return "invalid BIOS command"
	case 0x02:
		return "BIOS address mark not found"
	case 0x04:
		return "BIOS sector not found"
	case 0x08:
		return "BIOS DMA overrun"
	case 0x09:
		return "BIOS DMA crossed a 64 KiB boundary"
	case 0x10:
		return "BIOS data error"
	case 0x20:
		return "BIOS controller failure"
	case 0x40:
		return "BIOS seek failed"
	case 0x80:
		return "BIOS disk timed out"
	}
	return "BIOS error"
}

func result(regs *Registers) error {
	if regs.Flags&FlagCarry != 0 {
		return Error(regs.AX >> 8)
	}
	return nil
}

func nearPointer(data []byte) uint16 {
	if len(data) == 0 {
		return 0
	}
	return uint16(uintptr(unsafe.Pointer(&data[0])))
}

var diskPacketStorage [32]byte

// Interrupt invokes a supported firmware interrupt with explicit registers.
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
	case 0x17:
		interrupt17(regs)
	case 0x1a:
		interrupt1A(regs)
	}
}

// BootDrive returns the DL value with which the BIOS entered the boot sector.
func BootDrive() byte { return bootDrive() }

func SetVideoMode(mode byte) {
	regs := Registers{AX: uint16(mode)}
	interrupt10(&regs)
}

func Teletype(value byte) {
	regs := Registers{AX: 0x0e00 | uint16(value), BX: 7}
	interrupt10(&regs)
}

func WriteConsole(text string) {
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			Teletype('\r')
		}
		Teletype(text[i])
	}
}

func ReadKey() (ascii, scan byte) {
	regs := Registers{}
	interrupt16(&regs)
	return byte(regs.AX), byte(regs.AX >> 8)
}

func KeyAvailable() bool {
	regs := Registers{AX: 0x0100}
	interrupt16(&regs)
	return regs.Flags&FlagZero == 0
}

// Ticks returns the 18.2 Hz tick counter and midnight rollover byte.
func Ticks() (uint32, byte) {
	regs := Registers{}
	interrupt1A(&regs)
	return uint32(regs.CX)<<16 | uint32(regs.DX), byte(regs.AX)
}

func ResetDisk(drive byte) error {
	regs := Registers{DX: uint16(drive)}
	interrupt13(&regs)
	return result(&regs)
}

// ExtensionsAvailable reports support for the INT 13h extensions used by the
// generated boot sector and ReadSectors.
func ExtensionsAvailable(drive byte) bool {
	regs := Registers{AX: 0x4100, BX: 0x55aa, DX: uint16(drive)}
	interrupt13(&regs)
	return regs.Flags&FlagCarry == 0 && regs.BX == 0xaa55 && regs.CX&1 != 0
}

// ReadSectors reads count sectors using the EDD disk-address-packet service.
// The destination must fit without crossing a 64 KiB segment boundary.
func ReadSectors(drive byte, lba uint64, count, segment, offset uint16) error {
	if count == 0 || count > 127 || offset > uint16(0)-count*512 {
		return Error(0x09)
	}
	base := uintptr(unsafe.Pointer(&diskPacketStorage[0]))
	start := (16 - int(base&15)) & 15
	packet := diskPacketStorage[start : start+16]
	for i := range packet {
		packet[i] = 0
	}
	packet[0] = 16
	packet[2] = byte(count)
	packet[3] = byte(count >> 8)
	packet[4] = byte(offset)
	packet[5] = byte(offset >> 8)
	packet[6] = byte(segment)
	packet[7] = byte(segment >> 8)
	packet[8] = byte(lba)
	packet[9] = byte(lba >> 8)
	packet[10] = byte(lba >> 16)
	packet[11] = byte(lba >> 24)
	packet[12] = byte(lba >> 32)
	packet[13] = byte(lba >> 40)
	packet[14] = byte(lba >> 48)
	packet[15] = byte(lba >> 56)
	regs := Registers{AX: 0x4200, DX: uint16(drive), SI: nearPointer(packet)}
	interrupt13(&regs)
	return result(&regs)
}

// ReadCHS exposes the original cylinder/head/sector service for machines that
// do not implement EDD. Sector numbers are one-based.
func ReadCHS(drive byte, cylinder uint16, head, sector, count byte, segment, offset uint16) error {
	if cylinder > 1023 || sector == 0 || sector > 63 || count == 0 || count > 127 ||
		offset > uint16(0)-uint16(count)*512 {
		return Error(0x09)
	}
	cx := (cylinder&0xff)<<8 | (cylinder&0x300)>>2 | uint16(sector)
	regs := Registers{AX: 0x0200 | uint16(count), BX: offset, CX: cx,
		DX: uint16(head)<<8 | uint16(drive), ES: segment}
	interrupt13(&regs)
	return result(&regs)
}

func InitSerial(port, configuration byte) error {
	regs := Registers{AX: uint16(configuration), DX: uint16(port)}
	interrupt14(&regs)
	if byte(regs.AX>>8)&0x80 != 0 {
		return Error(0x80)
	}
	return nil
}

func WriteSerial(port, value byte) error {
	regs := Registers{AX: 0x0100 | uint16(value), DX: uint16(port)}
	interrupt14(&regs)
	if byte(regs.AX>>8)&0x80 != 0 {
		return Error(0x80)
	}
	return nil
}

func In8(port uint16) byte                   { return portIn8(port) }
func Out8(port uint16, value byte)           { portOut8(port, value) }
func Out16(port uint16, value uint16)        { portOut16(port, value) }
func Load8(segment, offset uint16) byte      { return segmentLoad8(segment, offset) }
func Store8(segment, offset uint16, v byte)  { segmentStore8(segment, offset, v) }
func Write(segment, offset uint16, b []byte) { segmentWrite(segment, offset, b) }

// EnterLongMode leaves BIOS real mode, identity maps the first GiB of physical
// memory, and transfers control to entry as 64-bit code. Entry must be an
// in-place freestanding/amd64 program compiled into this bios/8086 image. The
// transition disables interrupts and does not return.
func EnterLongMode(entry uintptr) { enterLongMode(entry) }
