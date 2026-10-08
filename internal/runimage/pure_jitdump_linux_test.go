//go:build !renvo && linux

package runimage

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestJITDumpExclusiveAndArenaLifecycle(t *testing.T) {
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		t.Skip("native host unavailable")
	}
	path := filepath.Join(t.TempDir(), "jit-test.dump")
	j, err := NewJITDump(path)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	if duplicate, err := NewJITDump(path); err == nil {
		duplicate.Close()
		t.Fatal("existing capture overwritten")
	}
	a, err := NewCodeArena(4096)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.SetJITWriter(j)
	code := []byte{0xc3}
	if runtime.GOARCH == "arm64" {
		code = []byte{0xc0, 0x03, 0x5f, 0xd6}
	}
	entry, err := a.InstallBlock(code, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.NameEntry(entry, "first"); err != nil {
		t.Fatal(err)
	}
	// Reject an interior address without emitting a fabricated code image.
	if a.NameEntry(entry+1, "interior") == nil {
		t.Fatal("unaligned interior image admitted")
	}
	if a.NameEntry(entry, "bad\x00symbol") == nil {
		t.Fatal("invalid NUL symbol admitted")
	}
	other, err := a.InstallBlock(code, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = a.NameEntry(other, "second"); err != nil {
		t.Fatal(err)
	}
	if err = a.Close(); err != nil {
		t.Fatal(err)
	}
	if err = j.Close(); err != nil {
		t.Fatal(err)
	}
	if j.WriteCode(1, code, "after-close") == nil {
		t.Fatal("closed stream accepted load")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 40 || binary.LittleEndian.Uint32(data) != 0x4a695444 || binary.LittleEndian.Uint32(data[4:]) != 1 || binary.LittleEndian.Uint32(data[8:]) != 40 {
		t.Fatal("invalid perf header")
	}
	wantFlags := uint64(0)
	if runtime.GOARCH == "amd64" {
		wantFlags = 1 // perf requires architecture timestamps for Intel PT.
	}
	if binary.LittleEndian.Uint64(data[32:40]) != wantFlags {
		t.Fatal("incorrect timestamp domain")
	}
	lastTimestamp := binary.LittleEndian.Uint64(data[24:])
	if lastTimestamp == 0 {
		t.Fatal("missing load timestamp")
	}
	loads, closed := 0, false
	for offset := 40; offset < len(data); {
		if len(data)-offset < 16 {
			t.Fatal("partial record header")
		}
		record := data[offset:]
		id, size := binary.LittleEndian.Uint32(record), int(binary.LittleEndian.Uint32(record[4:]))
		if size < 16 || size > len(record) {
			t.Fatal("partial record body")
		}
		timestamp := binary.LittleEndian.Uint64(record[8:])
		if timestamp < lastTimestamp {
			t.Fatal("non-monotonic load time")
		}
		lastTimestamp = timestamp
		if id == 0 {
			if size < 57 {
				t.Fatal("short code load")
			}
			nameEnd := bytes.IndexByte(record[56:size], 0)
			if nameEnd < 0 || binary.LittleEndian.Uint64(record[24:]) != binary.LittleEndian.Uint64(record[32:]) || !bytes.Equal(record[57+nameEnd:size], code) {
				t.Fatal("native bytes/name/address not reconstructible")
			}
			loads++
		} else if id == 3 {
			closed = true
		} else {
			t.Fatal("unexpected record", id)
		}
		offset += size
	}
	if loads != 2 || !closed {
		t.Fatal("incomplete lifecycle", loads, closed)
	}
}
