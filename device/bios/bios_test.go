package bios

import (
	"bytes"
	"testing"
	"unsafe"
)

func TestFirmwareServicesPreserveBIOSContracts(t *testing.T) {
	oldInterrupt, oldBoot := InterruptHook, BootDriveHook
	defer func() { InterruptHook, BootDriveHook = oldInterrupt, oldBoot }()
	BootDriveHook = func() byte { return 0x81 }
	if got := BootDrive(); got != 0x81 {
		t.Fatalf("BootDrive = %#x", got)
	}
	var vectors []byte
	InterruptHook = func(vector byte, regs *Registers) {
		vectors = append(vectors, vector)
		switch vector {
		case 0x13:
			if regs.AX == 0x4100 {
				regs.BX, regs.CX, regs.Flags = 0xaa55, 1, 0
			}
		case 0x16:
			if regs.AX == 0x0100 {
				regs.Flags &^= FlagZero
			} else {
				regs.AX = 0x1e61
			}
		case 0x1a:
			regs.CX, regs.DX, regs.AX = 0x1234, 0x5678, 2
		}
	}
	if !ExtensionsAvailable(0x81) {
		t.Fatal("EDD extension handshake was not recognized")
	}
	if !KeyAvailable() {
		t.Fatal("keyboard availability did not use the zero flag")
	}
	ascii, scan := ReadKey()
	if ascii != 'a' || scan != 0x1e {
		t.Fatalf("ReadKey = %#x/%#x", ascii, scan)
	}
	ticks, rollovers := Ticks()
	if ticks != 0x12345678 || rollovers != 2 {
		t.Fatalf("Ticks = %#x/%d", ticks, rollovers)
	}
	if len(vectors) != 4 || vectors[0] != 0x13 || vectors[1] != 0x16 || vectors[2] != 0x16 || vectors[3] != 0x1a {
		t.Fatalf("firmware vectors = %v", vectors)
	}
}

func TestDiskReadsRejectUnsafeTransfersAndReportFirmwareErrors(t *testing.T) {
	oldInterrupt := InterruptHook
	defer func() { InterruptHook = oldInterrupt }()
	calls := 0
	InterruptHook = func(vector byte, regs *Registers) {
		calls++
		if vector != 0x13 || regs.AX != 0x4200 || regs.DX != 0x80 || regs.SI == 0 {
			t.Fatalf("EDD registers = vector %#x, %+v", vector, *regs)
		}
		regs.AX = 0x0400
		regs.Flags = FlagCarry
	}
	for _, test := range []struct{ count, offset uint16 }{{0, 0}, {128, 0}, {2, 0xfe00}} {
		if err := ReadSectors(0x80, 1, test.count, 0x2000, test.offset); err == nil {
			t.Fatalf("unsafe transfer count=%d offset=%#x succeeded", test.count, test.offset)
		}
	}
	if calls != 0 {
		t.Fatalf("unsafe transfers reached firmware %d times", calls)
	}
	err := ReadSectors(0x80, 0x0123456789abcdef, 1, 0x2000, 0)
	if code, ok := err.(Error); !ok || code != 4 {
		t.Fatalf("firmware error = %#v", err)
	}
	if calls != 1 {
		t.Fatalf("safe transfer firmware calls = %d", calls)
	}
	base := uintptr(unsafe.Pointer(&diskPacketStorage[0]))
	start := (16 - int(base&15)) & 15
	packet := diskPacketStorage[start : start+16]
	wantLBA := []byte{0xef, 0xcd, 0xab, 0x89, 0x67, 0x45, 0x23, 0x01}
	if !bytes.Equal(packet[8:16], wantLBA) {
		t.Fatalf("EDD packet LBA = %x, want %x", packet[8:16], wantLBA)
	}
}

func TestEnterLongModeForwardsEntrypoint(t *testing.T) {
	oldHook := EnterLongModeHook
	defer func() { EnterLongModeHook = oldHook }()
	var got uintptr
	EnterLongModeHook = func(entry uintptr) { got = entry }
	EnterLongMode(0x1234)
	if got != 0x1234 {
		t.Fatalf("EnterLongMode entry = %#x", got)
	}
}
