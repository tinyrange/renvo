//go:build linux && amd64

package elflink

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLinksAndRunsMinimalObject(t *testing.T) {
	image, linkError := Link([]Input{{Name: "main.o", Data: minimalMainObject()}})
	if linkError.Message != "" {
		t.Fatalf("Link error = %#v", linkError)
	}
	path := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(path, image, 0755); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(path).CombinedOutput(); err != nil {
		t.Fatalf("linked executable: %v, output %q", err, output)
	}
}

func TestRejectsUndefinedEntry(t *testing.T) {
	object := minimalMainObject()
	copy(object[121:126], []byte("none\x00"))
	_, linkError := Link([]Input{{Name: "none.o", Data: object}})
	if linkError.Message != "entry function main is undefined" {
		t.Fatalf("error = %#v", linkError)
	}
}

func minimalMainObject() []byte {
	data := make([]byte, 384)
	copy(data, []byte{'\x7f', 'E', 'L', 'F', 2, 1, 1})
	binary.LittleEndian.PutUint16(data[16:], 1)
	binary.LittleEndian.PutUint16(data[18:], 62)
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint64(data[40:], 128)
	binary.LittleEndian.PutUint16(data[52:], 64)
	binary.LittleEndian.PutUint16(data[58:], 64)
	binary.LittleEndian.PutUint16(data[60:], 4)
	copy(data[64:], []byte{0x31, 0xc0, 0xc3})
	// The second symbol is global function main in section 1.
	binary.LittleEndian.PutUint32(data[96:], 1)
	data[100] = 0x12
	binary.LittleEndian.PutUint16(data[102:], 1)
	binary.LittleEndian.PutUint64(data[112:], 3)
	copy(data[120:], []byte("\x00main\x00"))
	section := func(index int) []byte { return data[128+index*64:] }
	text := section(1)
	binary.LittleEndian.PutUint32(text[4:], shtProgbits)
	binary.LittleEndian.PutUint64(text[8:], shfAlloc|shfExec)
	binary.LittleEndian.PutUint64(text[24:], 64)
	binary.LittleEndian.PutUint64(text[32:], 3)
	binary.LittleEndian.PutUint64(text[48:], 16)
	symbols := section(2)
	binary.LittleEndian.PutUint32(symbols[4:], shtSymtab)
	binary.LittleEndian.PutUint64(symbols[24:], 72)
	binary.LittleEndian.PutUint64(symbols[32:], 48)
	binary.LittleEndian.PutUint32(symbols[40:], 3)
	binary.LittleEndian.PutUint32(symbols[44:], 1)
	binary.LittleEndian.PutUint64(symbols[48:], 8)
	binary.LittleEndian.PutUint64(symbols[56:], 24)
	strings := section(3)
	binary.LittleEndian.PutUint32(strings[4:], 3)
	binary.LittleEndian.PutUint64(strings[24:], 120)
	binary.LittleEndian.PutUint64(strings[32:], 6)
	binary.LittleEndian.PutUint64(strings[48:], 1)
	return data
}
