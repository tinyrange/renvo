package main

import (
	"path/filepath"
	"testing"
)

func TestPlatformCatalogIncludesMiniScalesDriverAndExample(t *testing.T) {
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
}
