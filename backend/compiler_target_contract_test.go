package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type renvoAdvertisedTargetContract struct {
	name  string
	id    int
	magic string
}

var renvoAdvertisedTargetContracts = []renvoAdvertisedTargetContract{
	{"linux/amd64", renvoTargetLinuxAmd64, "\x7fELF"},
	{"linux/386", renvoTargetLinux386, "\x7fELF"},
	{"linux/aarch64", renvoTargetLinuxAarch64, "\x7fELF"},
	{"linux/arm", renvoTargetLinuxArm, "\x7fELF"},
	{"windows/amd64", renvoTargetWindowsAmd64, "MZ"},
	{"windows/386", renvoTargetWindows386, "MZ"},
	{"windows/arm64", renvoTargetWindowsArm64, "MZ"},
	{"wasi/wasm32", renvoTargetWasiWasm32, "\x00asm"},
	{"darwin/arm64", renvoTargetDarwinArm64, "\xcf\xfa\xed\xfe"},
}

func TestAdvertisedTargetsHaveProfilesAndRecognizableImages(t *testing.T) {
	source := []byte("package main\nfunc appMain() int { print(\"PASS\\n\"); return 0 }\n")
	for _, contract := range renvoAdvertisedTargetContracts {
		contract := contract
		t.Run(strings.ReplaceAll(contract.name, "/", "-"), func(t *testing.T) {
			if got := renvoParseTargetArg(contract.name); got != contract.id {
				t.Fatalf("target parser returned %d, want %d", got, contract.id)
			}
			profile, ok := renvoProfileForTarget(contract.id)
			if !ok || !renvoProfileIsValid(profile) {
				t.Fatalf("target profile invalid: %#v", profile)
			}
			image, ok := RenvoCompileSourceToBytesStrip(source, contract.name, true)
			if !ok {
				t.Fatal("target image compilation failed")
			}
			if len(image) < len(contract.magic) || string(image[:len(contract.magic)]) != contract.magic {
				t.Fatalf("image prefix = %x, want %x", image[:min(len(image), len(contract.magic))], []byte(contract.magic))
			}
		})
	}
}

func TestCompilerSourceManifestCoversBackendImplementationFiles(t *testing.T) {
	data, err := os.ReadFile("compiler_sources.txt")
	if err != nil {
		t.Fatalf("read compiler source manifest: %v", err)
	}
	var manifest []string
	for _, line := range strings.Split(string(data), "\n") {
		path := strings.TrimSpace(line)
		if path != "" {
			manifest = append(manifest, path)
		}
	}
	implementationFiles, err := filepath.Glob("compiler_*_impl.go")
	if err != nil {
		t.Fatalf("glob backend implementation files: %v", err)
	}
	implementationFiles = append(implementationFiles, "compiler_main.go")
	sort.Strings(manifest)
	sort.Strings(implementationFiles)
	if strings.Join(manifest, "\n") != strings.Join(implementationFiles, "\n") {
		t.Fatalf("compiler source manifest drift\nmanifest: %v\nfiles: %v", manifest, implementationFiles)
	}
}
