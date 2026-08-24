package backendjit

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/espflash"
)

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
	compile := func(source string) espflash.LoadImage {
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
		image, err := espflash.ParseLoadImage(result.Binary)
		if err != nil {
			t.Fatal(err)
		}
		return image
	}

	base := compile(hotReloadDeltaBaseSource)
	reordered := compile(hotReloadDeltaReorderedSource)
	assertHotReloadDelta(t, base, reordered, 16, "reordered calls")

	controlFlow := compile(hotReloadDeltaControlFlowSource)
	assertHotReloadDelta(t, reordered, controlFlow, 512, "small control-flow edit")
}

func assertHotReloadDelta(t *testing.T, previous espflash.LoadImage, next espflash.LoadImage, limit int, edit string) {
	t.Helper()
	patches := espflash.PlanPatches(&previous, next)
	bytes := 0
	for i := 0; i < len(patches); i++ {
		bytes += len(patches[i].Data)
	}
	t.Logf("%s delta: %d bytes in %d patches", edit, bytes, len(patches))
	if bytes == 0 || bytes > limit {
		t.Fatalf("%s writes %d bytes in %d patches, want 1..%d bytes", edit, bytes, len(patches), limit)
	}
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
