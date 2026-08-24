package main

import (
	"path/filepath"
	"testing"
)

func TestPlatformCatalogIncludesNanoC6UnitDriversAndExamples(t *testing.T) {
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	packages, err := buildPlatformPackages(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	driver, ok := packages["renvo.dev/device/sensor/miniscale"]
	if !ok || driver.Main || len(driver.Files) != 1 || driver.Files[0] != "miniscale.go" {
		t.Fatalf("Mini Scales driver catalog entry = %#v, present=%v", driver, ok)
	}
	example, ok := packages["renvo.dev/examples/m5nanoc6/miniscale"]
	if !ok || !example.Main || example.Target != "esp32c6/riscv32" || example.Board != "M5Stack NanoC6" ||
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
	synthExample, ok := packages["renvo.dev/examples/m5nanoc6/synth"]
	if !ok || !synthExample.Main || synthExample.Target != "esp32c6/riscv32" || synthExample.Board != "M5Stack NanoC6" ||
		len(synthExample.Files) != 1 || synthExample.Files[0] != "main.go" {
		t.Fatalf("Unit Synth example catalog entry = %#v, present=%v", synthExample, ok)
	}
}
