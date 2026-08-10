package linkedimage

import (
	"encoding/binary"
	"testing"
)

func TestPayloadRejectsMalformedHeaderAndChecksum(t *testing.T) {
	native := testELFImage()
	image := testTransport(native)
	target, format, payload, ok := Payload(image)
	if !ok || target != 1 || format != FormatELF || len(payload) != len(native) {
		t.Fatalf("Payload = %d, %d, %d, %v", target, format, len(payload), ok)
	}
	badHeader := append([]byte(nil), image...)
	badHeader[4] = 2
	if _, _, _, ok = Payload(badHeader); ok {
		t.Fatal("Payload accepted an unsupported version")
	}
	badChecksum := append([]byte(nil), image...)
	badChecksum[len(badChecksum)-1] ^= 1
	if _, _, _, ok = Payload(badChecksum); ok {
		t.Fatal("Payload accepted a corrupt payload")
	}
}

func TestLinuxLayoutNormalizesLoadSegments(t *testing.T) {
	native := testELFImage()
	entry, size, segments, ok := LinuxLayout(native)
	if !ok {
		t.Fatal("LinuxLayout rejected a valid image")
	}
	if entry != 176 || size != 8192 || len(segments) != 2 {
		t.Fatalf("LinuxLayout = entry %d, size %d, segments %#v", entry, size, segments)
	}
	if segments[0].Permissions != 5 || segments[0].FileSize != len(native) ||
		segments[1].Address != 4096 || segments[1].FileSize != 0 || segments[1].MemorySize != 4096 {
		t.Fatalf("segments = %#v", segments)
	}
}

func TestDarwinLayoutNormalizesMachSegmentPermissions(t *testing.T) {
	const (
		segmentSize = 72
		mainSize    = 24
		commands    = 3*segmentSize + mainSize
		fileStart   = 32 + commands
	)
	native := make([]byte, fileStart+3)
	copy(native, "\xcf\xfa\xed\xfe")
	binary.LittleEndian.PutUint32(native[16:20], 4)
	binary.LittleEndian.PutUint32(native[20:24], commands)
	protections := []uint32{5, 3, 1} // Mach: RX, RW, R.
	for i, protection := range protections {
		at := 32 + i*segmentSize
		binary.LittleEndian.PutUint32(native[at:at+4], 0x19)
		binary.LittleEndian.PutUint32(native[at+4:at+8], segmentSize)
		binary.LittleEndian.PutUint64(native[at+24:at+32], uint64(0x100000000+i*0x4000))
		binary.LittleEndian.PutUint64(native[at+32:at+40], 0x4000)
		binary.LittleEndian.PutUint64(native[at+40:at+48], uint64(fileStart+i))
		binary.LittleEndian.PutUint64(native[at+48:at+56], 1)
		binary.LittleEndian.PutUint32(native[at+60:at+64], protection)
	}
	main := 32 + 3*segmentSize
	binary.LittleEndian.PutUint32(native[main:main+4], 0x80000028)
	binary.LittleEndian.PutUint32(native[main+4:main+8], mainSize)
	binary.LittleEndian.PutUint64(native[main+8:main+16], fileStart)

	entry, memorySize, segments, _, _, ok := DarwinLayout(native)
	if !ok || entry != 0 || memorySize != 0xc000 || len(segments) != 3 {
		t.Fatalf("DarwinLayout = entry %#x, size %#x, segments %#v, ok %v",
			entry, memorySize, segments, ok)
	}
	want := []int{5, 6, 4} // Native/ELF convention: RX, RW, R.
	for i := range want {
		if segments[i].Permissions != want[i] {
			t.Fatalf("segment %d permissions = %d, want %d", i, segments[i].Permissions, want[i])
		}
	}
}

func TestPersistentSymbolsDecodeLinkTrailer(t *testing.T) {
	native := testELFImage()
	native = append(native,
		7, 0, 0, 0,
		0, 16, 0, 0,
		24, 0, 0, 0,
		32, 16, 0, 0,
		1, 0, 0, 0,
		'R', 'P', 'L', '1',
	)
	symbols, ok := PersistentSymbols(native, 8192)
	if !ok || len(symbols) != 1 {
		t.Fatalf("PersistentSymbols = %#v, %v", symbols, ok)
	}
	if symbols[0].ID != 7 || symbols[0].Address != 4096 ||
		symbols[0].Size != 24 || symbols[0].RestoreAddress != 4128 {
		t.Fatalf("symbol = %#v", symbols[0])
	}
}

func TestPersistentSymbolsRejectOutOfRangeStorage(t *testing.T) {
	native := testELFImage()
	native = append(native,
		1, 0, 0, 0,
		0xff, 0xff, 0, 0,
		8, 0, 0, 0,
		0, 16, 0, 0,
		1, 0, 0, 0,
		'R', 'P', 'L', '1',
	)
	if _, ok := PersistentSymbols(native, 8192); ok {
		t.Fatal("PersistentSymbols accepted an out-of-range slot")
	}
}

func TestBaseRelocationsDecodeBeforePersistentSymbols(t *testing.T) {
	native := testELFImage()
	native = append(native,
		32, 0, 0, 0,
		64, 0, 0, 0,
		2, 0, 0, 0,
		'R', 'B', 'R', '1',
		7, 0, 0, 0,
		0, 16, 0, 0,
		24, 0, 0, 0,
		32, 16, 0, 0,
		1, 0, 0, 0,
		'R', 'P', 'L', '1',
	)
	relocations, ok := BaseRelocations(native, 8192)
	if !ok || len(relocations) != 2 || relocations[0] != 32 || relocations[1] != 64 {
		t.Fatalf("BaseRelocations = %#v, %v", relocations, ok)
	}
}

func testTransport(native []byte) []byte {
	image := make([]byte, 20+len(native))
	copy(image, "RNVI")
	binary.LittleEndian.PutUint16(image[4:6], 1)
	binary.LittleEndian.PutUint16(image[6:8], 20)
	binary.LittleEndian.PutUint16(image[8:10], 1)
	image[10] = FormatELF
	binary.LittleEndian.PutUint32(image[12:16], uint32(len(native)))
	a, b := imageAdler(native)
	binary.LittleEndian.PutUint16(image[16:18], uint16(a))
	binary.LittleEndian.PutUint16(image[18:20], uint16(b))
	copy(image[20:], native)
	return image
}

func testELFImage() []byte {
	const codeOffset = 176
	data := make([]byte, codeOffset+1)
	copy(data, "\x7fELF")
	data[4], data[5], data[6] = 2, 1, 1
	binary.LittleEndian.PutUint16(data[16:18], 3)
	binary.LittleEndian.PutUint16(data[18:20], 62)
	binary.LittleEndian.PutUint32(data[20:24], 1)
	binary.LittleEndian.PutUint64(data[24:32], codeOffset)
	binary.LittleEndian.PutUint64(data[32:40], 64)
	binary.LittleEndian.PutUint16(data[52:54], 64)
	binary.LittleEndian.PutUint16(data[54:56], 56)
	binary.LittleEndian.PutUint16(data[56:58], 2)
	first := 64
	binary.LittleEndian.PutUint32(data[first:first+4], 1)
	binary.LittleEndian.PutUint32(data[first+4:first+8], 5)
	binary.LittleEndian.PutUint64(data[first+32:first+40], uint64(len(data)))
	binary.LittleEndian.PutUint64(data[first+40:first+48], uint64(len(data)))
	binary.LittleEndian.PutUint64(data[first+48:first+56], 4096)
	second := first + 56
	binary.LittleEndian.PutUint32(data[second:second+4], 1)
	binary.LittleEndian.PutUint32(data[second+4:second+8], 6)
	binary.LittleEndian.PutUint64(data[second+8:second+16], 4096)
	binary.LittleEndian.PutUint64(data[second+16:second+24], 4096)
	binary.LittleEndian.PutUint64(data[second+40:second+48], 4096)
	binary.LittleEndian.PutUint64(data[second+48:second+56], 4096)
	data[codeOffset] = 0xc3
	return data
}
