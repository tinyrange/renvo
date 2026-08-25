package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlatformCatalogIncludesNanoC6UnitDriversAndExamples(t *testing.T) {
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	boards, err := readBoardDefinitions(root)
	if err != nil {
		t.Fatal(err)
	}
	packages, err := buildPlatformPackages(root, t.TempDir(), boards)
	if err != nil {
		t.Fatal(err)
	}
	driver, ok := packages["renvo.dev/device/sensor/miniscale"]
	if !ok || driver.Main || len(driver.Files) != 1 || driver.Files[0] != "miniscale.go" {
		t.Fatalf("Mini Scales driver catalog entry = %#v, present=%v", driver, ok)
	}
	example, ok := packages["renvo.dev/examples/device/miniscale"]
	if !ok || !example.Main || len(example.Boards) != 1 || example.Boards[0].Name != "M5Stack NanoC6" ||
		len(example.Files) != 1 || example.Files[0] != "main.go" {
		t.Fatalf("Mini Scales example catalog entry = %#v, present=%v", example, ok)
	}
	synthDriver, ok := packages["renvo.dev/device/audio/sam2695"]
	if !ok || synthDriver.Main || len(synthDriver.Files) != 1 || synthDriver.Files[0] != "sam2695.go" {
		t.Fatalf("Unit Synth driver catalog entry = %#v, present=%v", synthDriver, ok)
	}
	uart, ok := packages["renvo.dev/device/uart"]
	if !ok || uart.Main || len(uart.Files) != 1 || uart.Files[0] != "uart.go" {
		t.Fatalf("UART catalog entry = %#v, present=%v", uart, ok)
	}
	synthExample, ok := packages["renvo.dev/examples/device/synth"]
	if !ok || !synthExample.Main || len(synthExample.Boards) != 1 || synthExample.Boards[0].Name != "M5Stack NanoC6" ||
		len(synthExample.Files) != 1 || synthExample.Files[0] != "main.go" {
		t.Fatalf("Unit Synth example catalog entry = %#v, present=%v", synthExample, ok)
	}
}

func TestVersionAssetUsesContentHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compiler.wasm")
	if err := os.WriteFile(path, []byte("compiler one"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := versionAsset(dir, "compiler.wasm")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(first, "compiler.wasm?v=") {
		t.Fatalf("versioned asset = %q", first)
	}
	if err := os.WriteFile(path, []byte("compiler two"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := versionAsset(dir, "compiler.wasm")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("content change retained version %q", first)
	}
}

func TestPDP11ExampleIsClassifiedAsAComputer(t *testing.T) {
	var found *platformPackageSpec
	for _, spec := range platformPackageSpecs(nil) {
		if spec.Path == "examples/pdp11v7" {
			copy := spec
			found = &copy
			break
		}
	}
	if found == nil || len(found.Computers) != 1 {
		t.Fatalf("PDP-11 computers = %#v", found)
	}
	if found.Computers[0].Name != "PDP-11" || found.Computers[0].Family != "Retro computer" {
		t.Fatalf("PDP-11 metadata = %#v", found.Computers[0])
	}
	if len(found.Boards) != 0 {
		t.Fatalf("PDP-11 boards = %#v", found.Boards)
	}
}
