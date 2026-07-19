package main

import (
	"encoding/binary"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"renvo.dev/backend/unit"
)

func TestBackendRenvoUnitGoldenVectors(t *testing.T) {
	for _, name := range []string{"v1-core.hex", "v1-full.hex"} {
		data := readBackendGolden(t, "unit/testdata/"+name)
		program, isUnit, ok := renvoDecodeUnitProgram(data)
		if !isUnit || !ok || !program.ok || string(program.src) != "package p\n" {
			t.Fatalf("backend failed to decode %s: isUnit=%v ok=%v programOK=%v", name, isUnit, ok, program.ok)
		}
	}
}

func TestBackendRenvoUnitVersionOneCompatibility(t *testing.T) {
	core := readBackendGolden(t, "unit/testdata/v1-core.hex")
	unknown := appendBackendUnitChild(core, 60000, []byte{1, 2, 3})
	if _, isUnit, ok := renvoDecodeUnitProgram(unknown); !isUnit || !ok {
		t.Fatal("backend rejected unknown optional tag")
	}

	version := append([]byte(nil), core...)
	binary.LittleEndian.PutUint16(version[4:6], unit.Version+1)
	if _, isUnit, ok := renvoDecodeUnitProgram(version); !isUnit || ok {
		t.Fatal("backend accepted unsupported version")
	}

	flags := append([]byte(nil), core...)
	binary.LittleEndian.PutUint16(flags[6:8], 1)
	if _, isUnit, ok := renvoDecodeUnitProgram(flags); !isUnit || ok {
		t.Fatal("backend accepted nonzero flags")
	}
}

func TestBackendRenvoUnitRejectsMissingDuplicateAndMalformedData(t *testing.T) {
	core := readBackendGolden(t, "unit/testdata/v1-core.hex")
	for _, item := range unit.WireSchemaTags {
		if item.Role != "child" || !item.Required {
			continue
		}
		without := removeBackendUnitChild(t, core, item.Number)
		if _, isUnit, ok := renvoDecodeUnitProgram(without); !isUnit || ok {
			t.Errorf("backend accepted missing required tag %s", item.Name)
		}
	}

	full := readBackendGolden(t, "unit/testdata/v1-full.hex")
	for _, item := range unit.WireSchemaTags {
		if item.Role != "child" {
			continue
		}
		payload := backendUnitChildPayload(t, full, item.Number)
		duplicate := appendBackendUnitChild(full, item.Number, payload)
		if _, isUnit, ok := renvoDecodeUnitProgram(duplicate); !isUnit || ok {
			t.Errorf("backend accepted duplicate tag %s", item.Name)
		}
	}
	for _, payload := range [][]byte{{0x80, 0x00}, {0xff, 0xff, 0xff, 0xff, 0x10}} {
		badTokens := replaceBackendUnitChild(t, core, unit.TagTokens, payload)
		if _, isUnit, ok := renvoDecodeUnitProgram(badTokens); !isUnit || ok {
			t.Errorf("backend accepted invalid token count varint %x", payload)
		}
	}

	badDecl := replaceBackendUnitChild(t, core, unit.TagDecls, []byte{1, renvoTokVar, 99, 1, 0, 1})
	if _, isUnit, ok := renvoDecodeUnitProgram(badDecl); !isUnit || ok {
		t.Fatal("backend accepted declaration outside text/token ranges")
	}

	for end := 4; end < len(core); end++ {
		if _, isUnit, ok := renvoDecodeUnitProgram(core[:end]); !isUnit || ok {
			t.Fatalf("backend accepted truncation at byte %d", end)
		}
	}
	overflow := append([]byte(nil), core...)
	binary.LittleEndian.PutUint32(overflow[10:14], ^uint32(0))
	if _, isUnit, ok := renvoDecodeUnitProgram(overflow); !isUnit || ok {
		t.Fatal("backend accepted overflowing root length")
	}
}

func readBackendGolden(t *testing.T, path string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := hex.DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func appendBackendUnitChild(data []byte, tag uint16, payload []byte) []byte {
	out := append([]byte(nil), data...)
	var header [6]byte
	binary.LittleEndian.PutUint16(header[0:2], tag)
	binary.LittleEndian.PutUint32(header[2:6], uint32(len(payload)))
	out = append(out, header[:]...)
	out = append(out, payload...)
	binary.LittleEndian.PutUint32(out[10:14], uint32(len(out)-14))
	return out
}

func removeBackendUnitChild(t *testing.T, data []byte, want uint16) []byte {
	t.Helper()
	out := append([]byte(nil), data[:14]...)
	pos := 14
	removed := false
	for pos < len(data) {
		tag := binary.LittleEndian.Uint16(data[pos : pos+2])
		length := int(binary.LittleEndian.Uint32(data[pos+2 : pos+6]))
		next := pos + 6 + length
		if tag == want && !removed {
			removed = true
		} else {
			out = append(out, data[pos:next]...)
		}
		pos = next
	}
	if !removed {
		t.Fatalf("golden vector does not contain tag %d", want)
	}
	binary.LittleEndian.PutUint32(out[10:14], uint32(len(out)-14))
	return out
}

func backendUnitChildPayload(t *testing.T, data []byte, want uint16) []byte {
	t.Helper()
	pos := 14
	for pos < len(data) {
		tag := binary.LittleEndian.Uint16(data[pos : pos+2])
		length := int(binary.LittleEndian.Uint32(data[pos+2 : pos+6]))
		start := pos + 6
		next := start + length
		if tag == want {
			return append([]byte(nil), data[start:next]...)
		}
		pos = next
	}
	t.Fatalf("golden vector does not contain tag %d", want)
	return nil
}

func replaceBackendUnitChild(t *testing.T, data []byte, want uint16, payload []byte) []byte {
	t.Helper()
	without := removeBackendUnitChild(t, data, want)
	return appendBackendUnitChild(without, want, payload)
}
