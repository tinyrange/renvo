package backendjit

import (
	"bytes"
	"encoding/binary"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

func compileRP2Test(t *testing.T, definition, target, output string) []byte {
	t.Helper()
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	definition = filepath.Join(root, definition)
	result := driver.CompileFromFS([]string{
		"-backend", definition, "-t", target, "-tags", "pico2",
		"-o", output, filepath.Join(root, "examples", "device", "blink"),
	}, root, filepath.Join(root, "std"), driver.OSFS{},
		New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
			backendJITTestCacheDir, backendcompiled.Backend{}))
	if !result.Ok {
		t.Fatalf("%s compile failed: %#v", target, result.Diagnostic)
	}
	return result.Binary
}

func TestCompiledInBootstrapCompilesDualFamilyRP2UF2(t *testing.T) {
	image := compileRP2Test(t, "backends/rp2.rtg", "rp2/thumb", "blink.uf2")
	if len(image) == 0 || len(image)%512 != 0 {
		t.Fatalf("UF2 length = %d, want nonzero whole blocks", len(image))
	}
	var rp2040, rp2350 []byte
	for at := 0; at < len(image); at += 512 {
		block := image[at : at+512]
		if binary.LittleEndian.Uint32(block[0:4]) != 0x0a324655 ||
			binary.LittleEndian.Uint32(block[4:8]) != 0x9e5d5157 ||
			binary.LittleEndian.Uint32(block[508:512]) != 0x0ab16f30 {
			t.Fatalf("invalid UF2 block at %#x", at)
		}
		family := binary.LittleEndian.Uint32(block[28:32])
		switch family {
		case 0xe48bff56:
			rp2040 = append(rp2040, block[32:288]...)
		case 0xe48bff59:
			rp2350 = append(rp2350, block[32:288]...)
		default:
			t.Fatalf("unexpected UF2 family %#x", family)
		}
	}
	if len(rp2040) == 0 || len(rp2040) != len(rp2350) ||
		!bytes.Equal(rp2040[256:], rp2350[256:]) {
		t.Fatal("RP2040 and RP2350 UF2 streams do not share an executable payload")
	}
	if binary.LittleEndian.Uint32(rp2350[80:84]) != 0xffffded3 ||
		binary.LittleEndian.Uint32(rp2350[92:96]) != 0x10000109 ||
		binary.LittleEndian.Uint32(rp2350[96:100]) != 0x20042000 {
		t.Fatalf("RP2350 boot metadata is invalid: % x", rp2350[80:108])
	}
	if binary.LittleEndian.Uint32(rp2040[256:260]) != 0x20042000 ||
		binary.LittleEndian.Uint32(rp2040[260:264]) != 0x10000109 {
		t.Fatalf("RP2040 vector table is invalid: % x", rp2040[256:264])
	}
	if got, want := binary.LittleEndian.Uint32(rp2040[252:256]), rp2040BootCRC(rp2040[:252]); got != want {
		t.Fatalf("RP2040 boot2 CRC = %#x, want %#x", got, want)
	}
	if !bytes.Equal(rp2040[:8], []byte{0x00, 0xb5, 0x32, 0x4b, 0x21, 0x20, 0x58, 0x60}) {
		t.Fatalf("RP2040 W25Q080 boot2 prefix is invalid: % x", rp2040[:8])
	}
}

func rp2040BootCRC(data []byte) uint32 {
	crc := uint32(0xffffffff)
	for _, value := range data {
		var reversed uint32
		for bit := uint(0); bit < 8; bit++ {
			reversed |= uint32(value>>bit&1) << (7 - bit)
		}
		crc ^= reversed
		for bit := 0; bit < 8; bit++ {
			if crc&1 != 0 {
				crc = crc>>1 ^ 0xedb88320
			} else {
				crc >>= 1
			}
		}
	}
	var result uint32
	for bit := uint(0); bit < 32; bit++ {
		result |= crc >> bit & 1 << (31 - bit)
	}
	return result
}

func TestCompiledInBootstrapCompilesRP2DebugELF(t *testing.T) {
	image := compileRP2Test(t, "backends/rp2_debug.rtg", "rp2-debug/thumb", "blink.elf")
	if len(image) < 116 || !bytes.Equal(image[:4], []byte{0x7f, 'E', 'L', 'F'}) {
		t.Fatalf("debug output is not ELF32: % x", image[:minInt(4, len(image))])
	}
	if machine := binary.LittleEndian.Uint16(image[18:20]); machine != 40 {
		t.Fatalf("ELF machine = %d, want ARM (40)", machine)
	}
	if entry := binary.LittleEndian.Uint32(image[24:28]); entry != 0x20020101 {
		t.Fatalf("ELF entry = %#x, want SRAM Thumb entry 0x20020101", entry)
	}
	if text := binary.LittleEndian.Uint32(image[60:64]); text != 0x20020000 {
		t.Fatalf("text address = %#x, want 0x20020000", text)
	}
}
