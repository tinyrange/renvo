package unit

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func goldenProgram() Program {
	return Program{
		Package:    "p",
		ImportPath: "example/p",
		Text:       []byte("package p\n"),
		Tokens: []byte{
			renvoTokPackage, 0, 0, 0, 7, 0, 1, 0,
			renvoTokIdent, 8, 0, 0, 1, 0, 1, 0,
			renvoTokEOF, 10, 0, 0, 0, 0, 2, 0,
		},
	}
}

func TestFullGoldenVector(t *testing.T) {
	data, err := Marshal(goldenProgram())
	if err != nil {
		t.Fatal(err)
	}
	want := readGolden(t, "testdata/v1-full.hex")
	if !bytes.Equal(data, want) {
		t.Fatalf("host encoder drift\ngot  %x\nwant %x", data, want)
	}
}

func TestVersionOneGoldenVectors(t *testing.T) {
	for _, name := range []string{"v1-core.hex", "v1-full.hex"} {
		data := readGolden(t, "testdata/"+name)
		program, err := Unmarshal(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if program.Package != "p" || program.ImportPath != "example/p" || string(program.Text) != "package p\n" {
			t.Fatalf("%s decoded incorrectly: %#v", name, program)
		}
	}
}

func TestVersionOneCompatibilityPolicy(t *testing.T) {
	core := readGolden(t, "testdata/v1-core.hex")

	withUnknown := appendUnitChild(core, 60000, []byte{1, 2, 3})
	if _, err := Unmarshal(withUnknown); err != nil {
		t.Fatalf("unknown optional tag rejected: %v", err)
	}

	newVersion := append([]byte(nil), core...)
	binary.LittleEndian.PutUint16(newVersion[4:6], Version+1)
	if _, err := Unmarshal(newVersion); err == nil {
		t.Fatal("unsupported version accepted")
	}

	flags := append([]byte(nil), core...)
	binary.LittleEndian.PutUint16(flags[6:8], 1)
	if _, err := Unmarshal(flags); err == nil {
		t.Fatal("nonzero header flags accepted")
	}
}

func TestVersionOneRejectsMissingAndDuplicateTags(t *testing.T) {
	core := readGolden(t, "testdata/v1-core.hex")
	for _, item := range WireSchemaTags {
		if item.Role != "child" || !item.Required {
			continue
		}
		without := removeUnitChild(t, core, item.Number)
		if _, err := Unmarshal(without); err == nil {
			t.Errorf("missing required tag %s accepted", item.Name)
		}
	}

	full := readGolden(t, "testdata/v1-full.hex")
	for _, item := range WireSchemaTags {
		if item.Role != "child" {
			continue
		}
		payload := unitChildPayload(t, full, item.Number)
		duplicate := appendUnitChild(full, item.Number, payload)
		if _, err := Unmarshal(duplicate); err == nil {
			t.Errorf("duplicate tag %s accepted", item.Name)
		}
	}
}

func TestVersionOneRejectsTruncationAndOverflow(t *testing.T) {
	core := readGolden(t, "testdata/v1-core.hex")
	for end := 0; end < len(core); end++ {
		if _, err := Unmarshal(core[:end]); err == nil {
			t.Fatalf("truncated unit accepted at byte %d", end)
		}
	}
	overflow := append([]byte(nil), core...)
	binary.LittleEndian.PutUint32(overflow[10:14], ^uint32(0))
	if _, err := Unmarshal(overflow); err == nil {
		t.Fatal("overflowing root length accepted")
	}
}

func TestVersionOneRejectsMalformedCoreRanges(t *testing.T) {
	core := readGolden(t, "testdata/v1-core.hex")
	for _, payload := range [][]byte{{0x80, 0x00}, {0xff, 0xff, 0xff, 0xff, 0x10}} {
		badTokens := replaceUnitChild(t, core, TagTokens, payload)
		if _, err := Unmarshal(badTokens); err == nil {
			t.Errorf("invalid token count varint %x accepted", payload)
		}
	}
	badDecl := replaceUnitChild(t, core, TagDecls, []byte{1, renvoTokVar, 99, 1, 0, 1})
	if _, err := Unmarshal(badDecl); err == nil {
		t.Fatal("declaration outside text/token ranges accepted")
	}
	badFunc := replaceUnitChild(t, core, TagFuncs, []byte{1, 99, 1, 0, 0, 0, 0, 0, 0, 0})
	if _, err := Unmarshal(badFunc); err == nil {
		t.Fatal("function outside text/token ranges accepted")
	}
}

func readGolden(t *testing.T, path string) []byte {
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

func appendUnitChild(data []byte, tag uint16, payload []byte) []byte {
	out := append([]byte(nil), data...)
	var header [6]byte
	binary.LittleEndian.PutUint16(header[0:2], tag)
	binary.LittleEndian.PutUint32(header[2:6], uint32(len(payload)))
	out = append(out, header[:]...)
	out = append(out, payload...)
	binary.LittleEndian.PutUint32(out[10:14], uint32(len(out)-14))
	return out
}

func removeUnitChild(t *testing.T, data []byte, want uint16) []byte {
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

func unitChildPayload(t *testing.T, data []byte, want uint16) []byte {
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

func replaceUnitChild(t *testing.T, data []byte, want uint16, payload []byte) []byte {
	t.Helper()
	without := removeUnitChild(t, data, want)
	return appendUnitChild(without, want, payload)
}
