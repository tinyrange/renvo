package backendjit

import (
	"bytes"
	"debug/elf"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

type hotReloadSegment struct {
	address uint32
	data    []byte
}

func TestESP32C6JTAGLayoutKeepsEditDeltasLocal(t *testing.T) {
	if hostTarget() == "" {
		t.Skipf("no in-process prepared backend for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	project := t.TempDir()
	if err = os.WriteFile(filepath.Join(project, "go.mod"),
		[]byte("module example.com/hotreloaddelta\n\ngo 1.23\n"), 0644); err != nil {
		t.Fatal(err)
	}
	definition := filepath.Join(root, "backends", "esp32c6_jtag.rtg")
	backend := New(definition, filepath.Join(root, "backend"), filepath.Join(root, "std"),
		backendJITTestCacheDir, backendcompiled.Backend{})
	compile := func(source string) []hotReloadSegment {
		t.Helper()
		if err := os.WriteFile(filepath.Join(project, "main.go"), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
		result := driver.CompileFromFS([]string{
			"-backend", definition,
			"-t", "esp32c6-jtag/riscv32",
			"-s", "-o", "hotreload.elf", project,
		}, project, filepath.Join(root, "std"), driver.OSFS{}, backend)
		if !result.Ok {
			t.Fatalf("ESP32-C6 JTAG compile failed: %#v", result.Diagnostic)
		}
		image, err := elf.NewFile(bytes.NewReader(result.Binary))
		if err != nil {
			t.Fatal(err)
		}
		var segments []hotReloadSegment
		for _, program := range image.Progs {
			if program.Type != elf.PT_LOAD || program.Filesz == 0 {
				continue
			}
			data := make([]byte, program.Filesz)
			if _, err = program.ReadAt(data, 0); err != nil {
				t.Fatal(err)
			}
			segments = append(segments, hotReloadSegment{address: uint32(program.Vaddr), data: data})
		}
		return segments
	}

	base := compile(hotReloadDeltaBaseSource)
	reordered := compile(hotReloadDeltaReorderedSource)
	assertHotReloadDelta(t, base, reordered, 16, "reordered calls")

	controlFlow := compile(hotReloadDeltaControlFlowSource)
	assertHotReloadDelta(t, reordered, controlFlow, 512, "small control-flow edit")
}

func assertHotReloadDelta(t *testing.T, previous, next []hotReloadSegment, limit int, edit string) {
	t.Helper()
	changed, patches := hotReloadPatchSize(previous, next)
	t.Logf("%s delta: %d bytes in %d patches", edit, changed, patches)
	if changed == 0 || changed > limit {
		t.Fatalf("%s writes %d bytes in %d patches, want 1..%d bytes", edit, changed, patches, limit)
	}
}

func hotReloadPatchSize(previous, next []hotReloadSegment) (int, int) {
	bytes, patches := 0, 0
	for _, segment := range next {
		start, lastChanged := -1, -1
		for offset := 0; offset < (len(segment.data)+3)&^3; offset += 4 {
			changed := false
			for index := 0; index < 4; index++ {
				at := offset + index
				value := byte(0)
				if at < len(segment.data) {
					value = segment.data[at]
				}
				old, found := hotReloadByte(previous, segment.address+uint32(at))
				changed = changed || !found || old != value
			}
			if changed {
				if start < 0 {
					start = offset
				} else if offset-lastChanged > 20 {
					bytes += lastChanged + 4 - start
					patches++
					start = offset
				}
				lastChanged = offset
			}
		}
		if start >= 0 {
			bytes += lastChanged + 4 - start
			patches++
		}
	}
	return bytes, patches
}

func hotReloadByte(segments []hotReloadSegment, address uint32) (byte, bool) {
	for _, segment := range segments {
		size := uint32((len(segment.data) + 3) &^ 3)
		if address < segment.address || address >= segment.address+size {
			continue
		}
		offset := int(address - segment.address)
		if offset < len(segment.data) {
			return segment.data[offset], true
		}
		return 0, true
	}
	return 0, false
}

const hotReloadDeltaBaseSource = `package main

var sink int

func first() {
	sink = firstLeaf(sink)
}

func firstLeaf(value int) int {
	return value + 1
}

func second() {
	sink = secondLeaf(sink)
}

func secondLeaf(value int) int {
	return value * 7
}

func main() {
	first()
	second()
}
`

const hotReloadDeltaReorderedSource = `package main

var sink int

func first() {
	sink = firstLeaf(sink)
}

func firstLeaf(value int) int {
	return value + 1
}

func second() {
	sink = secondLeaf(sink)
}

func secondLeaf(value int) int {
	return value * 7
}

func main() {
	second()
	first()
}
`

const hotReloadDeltaControlFlowSource = `package main

var sink int

func first() {
	if sink > 0 {
		sink = firstLeaf(sink)
	} else {
		sink = firstLeaf(sink + 1)
	}
}

func firstLeaf(value int) int {
	return value + 1
}

func second() {
	sink = secondLeaf(sink)
}

func secondLeaf(value int) int {
	return value * 7
}

func main() {
	second()
	first()
}
`
