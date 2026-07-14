//go:build !rtg

package unit

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

func conformanceProgram() Program {
	return Program{
		Package:    "p",
		ImportPath: "example/p",
		Text:       []byte("package p\n"),
		Tokens: []Token{
			{Kind: TokenPackage, Start: 0, Size: 7, Line: 1},
			{Kind: TokenIdent, Start: 8, Size: 1, Line: 1},
			{Kind: TokenEOF, Start: 10, Size: 0, Line: 2},
		},
	}
}

func TestFullCodecGoldenVectors(t *testing.T) {
	full := readConformanceGolden(t, "../../../rtgunit/testdata/v1-full.hex")
	encoded, ok := Marshal(conformanceProgram())
	if !ok {
		t.Fatal("Marshal failed")
	}
	if !bytes.Equal(encoded, full) {
		t.Fatalf("full encoder drift\ngot  %x\nwant %x", encoded, full)
	}
	for _, name := range []string{"v1-core.hex", "v1-full.hex"} {
		data := readConformanceGolden(t, "../../../rtgunit/testdata/"+name)
		program, ok := Unmarshal(data)
		if !ok || program.Package != "p" || program.ImportPath != "example/p" || string(program.Text) != "package p\n" {
			t.Fatalf("failed to decode %s", name)
		}
	}
}

func TestFullCodecSkipsUnknownVersionOneTag(t *testing.T) {
	data := readConformanceGolden(t, "../../../rtgunit/testdata/v1-core.hex")
	var header [6]byte
	binary.LittleEndian.PutUint16(header[0:2], 60000)
	binary.LittleEndian.PutUint32(header[2:6], 2)
	data = append(data, header[:]...)
	data = append(data, 1, 2)
	binary.LittleEndian.PutUint32(data[10:14], uint32(len(data)-14))
	if _, ok := Unmarshal(data); !ok {
		t.Fatal("unknown optional tag rejected")
	}
}

func TestFullCodecRejectsInvalidVarints(t *testing.T) {
	core := readConformanceGolden(t, "../../../rtgunit/testdata/v1-core.hex")
	for _, payload := range [][]byte{{0x80, 0x00}, {0xff, 0xff, 0xff, 0xff, 0x10}} {
		data := replaceConformanceChild(t, core, TagTokens, payload)
		if _, ok := Unmarshal(data); ok {
			t.Errorf("accepted invalid token count varint %x", payload)
		}
	}
}

func readConformanceGolden(t *testing.T, path string) []byte {
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

func replaceConformanceChild(t *testing.T, data []byte, want int, payload []byte) []byte {
	t.Helper()
	out := append([]byte(nil), data[:14]...)
	pos := 14
	replaced := false
	for pos < len(data) {
		tag := int(binary.LittleEndian.Uint16(data[pos : pos+2]))
		length := int(binary.LittleEndian.Uint32(data[pos+2 : pos+6]))
		next := pos + 6 + length
		if tag == want && !replaced {
			var header [6]byte
			binary.LittleEndian.PutUint16(header[0:2], uint16(want))
			binary.LittleEndian.PutUint32(header[2:6], uint32(len(payload)))
			out = append(out, header[:]...)
			out = append(out, payload...)
			replaced = true
		} else {
			out = append(out, data[pos:next]...)
		}
		pos = next
	}
	if !replaced {
		t.Fatalf("golden vector does not contain tag %d", want)
	}
	binary.LittleEndian.PutUint32(out[10:14], uint32(len(out)-14))
	return out
}
