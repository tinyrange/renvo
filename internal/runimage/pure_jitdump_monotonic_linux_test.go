//go:build !renvo && linux

package runimage

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestJITDumpMonotonicClockDomain(t *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("native host unavailable")
	}
	before, err := jitMonotonicTimestamp()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "mono.dump")
	dump, err := NewJITDumpMonotonic(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = dump.WriteCode(4096, []byte{0}, "clock-domain"); err != nil {
		t.Fatal(err)
	}
	if err = dump.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := jitMonotonicTimestamp()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint64(data[32:40]) != 0 {
		t.Fatal("monotonic stream marked architecture clock")
	}
	stamps := []uint64{binary.LittleEndian.Uint64(data[24:32])}
	for offset := 40; offset < len(data); {
		if len(data)-offset < 16 {
			t.Fatal("partial timestamp record")
		}
		size := int(binary.LittleEndian.Uint32(data[offset+4:]))
		if size < 16 || size > len(data)-offset {
			t.Fatal("invalid record size")
		}
		stamps = append(stamps, binary.LittleEndian.Uint64(data[offset+8:]))
		offset += size
	}
	if len(stamps) != 3 {
		t.Fatal("missing header/load/close timestamps")
	}
	for _, stamp := range stamps {
		if stamp < before || stamp > after {
			t.Fatalf("timestamp %d outside monotonic interval [%d,%d]", stamp, before, after)
		}
		before = stamp
	}
}
