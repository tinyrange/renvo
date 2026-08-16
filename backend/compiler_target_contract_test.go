package main

import (
	"os"
	"path/filepath"
	"renvo.dev/internal/targetinfo"
	"sort"
	"strings"
	"testing"
)

type renvoAdvertisedTargetContract struct {
	name  string
	id    int
	magic string
	os    string
	isa   string
	word  int
}

func renvoAdvertisedTargetContracts(t *testing.T) []renvoAdvertisedTargetContract {
	t.Helper()
	var contracts []renvoAdvertisedTargetContract
	for _, descriptor := range targetinfo.All() {
		if !descriptor.Advertised || descriptor.Virtual {
			continue
		}
		magic := ""
		switch descriptor.Image {
		case "elf":
			magic = "\x7fELF"
		case "pe":
			magic = "MZ"
		case "wasm":
			magic = "\x00asm"
		case "rnvm":
			magic = "RNVB"
		case "mach-o":
			magic = "\xcf\xfa\xed\xfe"
		default:
			t.Fatalf("advertised target %s has unknown image format %s", descriptor.Name, descriptor.Image)
		}
		contracts = append(contracts, renvoAdvertisedTargetContract{
			name:  descriptor.Name,
			id:    renvoParseTargetArg(descriptor.Backend),
			magic: magic,
			os:    descriptor.OS,
			isa:   descriptor.ISA,
			word:  descriptor.WordBits,
		})
	}
	return contracts
}

func TestAdvertisedTargetsHaveProfilesAndRecognizableImages(t *testing.T) {
	source := []byte("package main\nfunc appMain() int { print(\"PASS\\n\"); return 0 }\n")
	seen := make(map[int]bool)
	for _, contract := range renvoAdvertisedTargetContracts(t) {
		contract := contract
		if contract.id == 0 || seen[contract.id] {
			t.Fatalf("target %s has invalid or duplicate backend ID %d", contract.name, contract.id)
		}
		seen[contract.id] = true
		t.Run(strings.ReplaceAll(contract.name, "/", "-"), func(t *testing.T) {
			if got := renvoParseTargetArg(contract.name); got != contract.id {
				t.Fatalf("target parser returned %d, want %d", got, contract.id)
			}
			profile, ok := renvoProfileForTarget(contract.id)
			if !ok || !renvoProfileIsValid(profile) {
				t.Fatalf("target profile invalid: %#v", profile)
			}
			wantOS := map[string]int{
				"linux": renvoOSLinux, "windows": renvoOSWindows, "darwin": renvoOSDarwin,
				"wasi": renvoOSWasi, "vm": renvoOSVM, "freebsd": renvoOSFreeBSD,
				"openbsd": renvoOSOpenBSD,
				"netbsd":  renvoOSNetBSD,
			}[contract.os]
			wantArch := map[string]int{
				"amd64": renvoArchAmd64, "386": renvoArch386, "aarch64": renvoArchAarch64,
				"arm": renvoArchArm, "wasm32": renvoArchWasm32, "vm32": renvoArchWasm32,
			}[contract.isa]
			if profile.os != wantOS || profile.arch != wantArch || profile.intBits != contract.word {
				t.Fatalf("generated backend projection = os:%d arch:%d word:%d, registry wants os:%d arch:%d word:%d",
					profile.os, profile.arch, profile.intBits, wantOS, wantArch, contract.word)
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
	// The statically prepared LLVM projection is selected only by the
	// renvo_prepared build tag. It must not enter the ordinary bootstrap source
	// manifest, whose stage compilers consume sources without host Go's file
	// selection pass.
	for i := 0; i < len(implementationFiles); i++ {
		if implementationFiles[i] == "compiler_llvm_prepared_impl.go" {
			implementationFiles = append(implementationFiles[:i], implementationFiles[i+1:]...)
			break
		}
	}
	implementationFiles = append(implementationFiles, "compiler_main.go")
	sort.Strings(manifest)
	sort.Strings(implementationFiles)
	if strings.Join(manifest, "\n") != strings.Join(implementationFiles, "\n") {
		t.Fatalf("compiler source manifest drift\nmanifest: %v\nfiles: %v", manifest, implementationFiles)
	}
}
