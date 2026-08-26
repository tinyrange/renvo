package dos

import "testing"

func resetHooks() {
	InterruptHook = nil
	PortInHook = nil
	PortOutHook = nil
}

func TestDateTimeAndKeyboardUseBIOSDOSContracts(t *testing.T) {
	t.Cleanup(resetHooks)
	InterruptHook = func(vector byte, regs *Registers) {
		if vector == 0x21 && regs.AX == 0x2a00 {
			regs.CX, regs.DX, regs.AX = 2026, 8<<8|26, 3
		} else if vector == 0x21 && regs.AX == 0x2c00 {
			regs.CX, regs.DX = 14<<8|35, 9<<8|42
		} else if vector == 0x16 && regs.AX == 0x0100 {
			regs.Flags &^= FlagZero
		} else if vector == 0x16 {
			regs.AX = 0x1e41
		}
	}
	now := Now()
	if now.Year != 2026 || now.Month != 8 || now.Day != 26 || now.Hour != 14 || now.Hundredth != 42 {
		t.Fatalf("Now = %#v", now)
	}
	if !KeyAvailable() || ReadKey() != (Key{ASCII: 'A', Scan: 0x1e}) {
		t.Fatal("keyboard BIOS contract was not preserved")
	}
}

func TestParseDirectoryEntry(t *testing.T) {
	dta := make([]byte, 43)
	dta[21] = AttributeDirectory | AttributeArchive
	dta[22], dta[23] = 0x34, 0x12
	dta[24], dta[25] = 0x78, 0x56
	dta[26], dta[27], dta[28], dta[29] = 0x78, 0x56, 0x34, 0x12
	copy(dta[30:], []byte("GAMES"))
	entry := parseDirEntry(dta)
	if entry.Name != "GAMES" || !entry.IsDir() || entry.Size != 0x12345678 || entry.Time != 0x1234 || entry.Date != 0x5678 {
		t.Fatalf("entry = %#v", entry)
	}
}

func TestFinderIgnoresUndefinedSetDTAFlags(t *testing.T) {
	t.Cleanup(resetHooks)
	var calls int
	InterruptHook = func(vector byte, regs *Registers) {
		if vector != 0x21 {
			t.Fatalf("interrupt = %#x", vector)
		}
		calls++
		if regs.AX == 0x1a00 {
			regs.Flags |= FlagCarry
			return
		}
		if regs.AX != 0x4e00 {
			t.Fatalf("function = %#x", regs.AX)
		}
		regs.Flags &^= FlagCarry
	}
	entry, ok, err := Find("*.TXT", AttributeDirectory).Next()
	if err != nil || !ok || entry.Name != "" || calls != 2 {
		t.Fatalf("Next = %#v, %v, %v; calls = %d", entry, ok, err, calls)
	}
}

func TestVGAPaletteAndSpeakerPorts(t *testing.T) {
	t.Cleanup(resetHooks)
	type write struct {
		port, value uint16
		width       int
	}
	var writes []write
	PortInHook = func(port uint16) byte { return 0 }
	PortOutHook = func(port uint16, value uint16, width int) { writes = append(writes, write{port, value, width}) }
	(&VGA13{}).Palette(7, 255, 128, 0)
	SpeakerOn(440)
	if len(writes) != 8 || writes[0].port != 0x3c8 || writes[1].value != 63 || writes[5].port != 0x42 || writes[7].port != 0x61 {
		t.Fatalf("port writes = %#v", writes)
	}
	if got := speakerDivisor(440); got != 2711 {
		t.Fatalf("440 Hz divisor = %d", got)
	}
}

func TestFileHandlesUseDOSContracts(t *testing.T) {
	t.Cleanup(resetHooks)
	var calls []Registers
	InterruptHook = func(vector byte, regs *Registers) {
		if vector != 0x21 {
			t.Fatalf("interrupt = %#x", vector)
		}
		calls = append(calls, *regs)
		switch regs.AX >> 8 {
		case 0x3c, 0x3d:
			regs.AX = 7
		case 0x3f, 0x40:
			regs.AX = regs.CX
		}
	}
	file, err := CreateFile("TEST.TXT", AttributeArchive)
	if err != nil || file.Handle != 7 {
		t.Fatalf("CreateFile = %#v, %v", file, err)
	}
	if written, writeErr := file.Write([]byte("abc")); written != 3 || writeErr != nil {
		t.Fatalf("Write = %d, %v", written, writeErr)
	}
	if read, readErr := file.Read(make([]byte, 4)); read != 4 || readErr != nil {
		t.Fatalf("Read = %d, %v", read, readErr)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 4 || calls[0].AX != 0x3c00 || calls[0].CX != AttributeArchive ||
		calls[1].AX != 0x4000 || calls[1].BX != 7 || calls[2].AX != 0x3f00 || calls[3].AX != 0x3e00 {
		t.Fatalf("DOS file calls = %#v", calls)
	}
}
