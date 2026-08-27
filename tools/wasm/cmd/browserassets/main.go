package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"renvo.dev/internal/backenddef"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/targetinfo"
)

type targetAsset struct {
	Name              string                  `json:"name"`
	Label             string                  `json:"label,omitempty"`
	FrontendTarget    string                  `json:"frontendTarget,omitempty"`
	BackendTarget     string                  `json:"backendTarget"`
	Backend           string                  `json:"backend"`
	BackendFormat     string                  `json:"backendFormat,omitempty"`
	RTGDefinition     string                  `json:"rtgDefinition,omitempty"`
	RTGDefinitionName string                  `json:"rtgDefinitionName,omitempty"`
	RTGImports        []targetDefinitionAsset `json:"rtgImports,omitempty"`
	CBackend          string                  `json:"cBackend,omitempty"`
	Output            string                  `json:"output"`
	Runnable          bool                    `json:"runnable,omitempty"`
	Device            string                  `json:"device,omitempty"`
	Docs              string                  `json:"docs,omitempty"`
	Artwork           string                  `json:"artwork,omitempty"`
	Tags              []string                `json:"tags,omitempty"`
	Definition        string                  `json:"definition,omitempty"`
	DescriptorVersion int                     `json:"descriptorVersion,omitempty"`
	Hidden            bool                    `json:"hidden,omitempty"`
}

type targetDefinitionAsset struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

type targetCatalog struct {
	Compiler        string        `json:"compiler"`
	Linker          string        `json:"linker"`
	LanguageService string        `json:"languageService"`
	Formatter       string        `json:"formatter"`
	BackendJIT      string        `json:"backendJIT"`
	VMBackend       string        `json:"vmBackend"`
	BrowserPrefix   string        `json:"browserPrefix"`
	BrowserSuffix   string        `json:"browserSuffix"`
	Stdlib          string        `json:"stdlib"`
	Targets         []targetAsset `json:"targets"`
}

type standardPackage struct {
	Files     []string         `json:"files"`
	Imports   []string         `json:"imports,omitempty"`
	Root      string           `json:"root,omitempty"`
	Main      bool             `json:"main,omitempty"`
	Target    string           `json:"target,omitempty"`
	ArenaSize int              `json:"arenaSize,omitempty"`
	Board     string           `json:"board,omitempty"`
	Boards    []boardTarget    `json:"boards,omitempty"`
	Computers []computerTarget `json:"computers,omitempty"`
	Language  string           `json:"language,omitempty"`
	Docs      *packageDocs     `json:"docs,omitempty"`
}

type docEntry struct {
	Name      string       `json:"name"`
	Signature string       `json:"signature"`
	Doc       string       `json:"doc,omitempty"`
	File      string       `json:"file,omitempty"`
	Line      int          `json:"line,omitempty"`
	Examples  []docExample `json:"examples,omitempty"`
}

type docExample struct {
	Name      string `json:"name,omitempty"`
	Doc       string `json:"doc,omitempty"`
	Code      string `json:"code"`
	Output    string `json:"output,omitempty"`
	Unordered bool   `json:"unordered,omitempty"`
}

type docType struct {
	docEntry
	Methods []docEntry `json:"methods,omitempty"`
}

type packageDocs struct {
	Name       string       `json:"name"`
	ImportPath string       `json:"importPath"`
	Doc        string       `json:"doc,omitempty"`
	Constants  []docEntry   `json:"constants,omitempty"`
	Variables  []docEntry   `json:"variables,omitempty"`
	Functions  []docEntry   `json:"functions,omitempty"`
	Types      []docType    `json:"types,omitempty"`
	Examples   []docExample `json:"examples,omitempty"`
}

type boardTarget struct {
	Name     string                `json:"name"`
	Target   string                `json:"target"`
	Docs     string                `json:"docs,omitempty"`
	Artwork  string                `json:"artwork,omitempty"`
	Hardware []hardwareRequirement `json:"hardware,omitempty"`
}

type computerTarget struct {
	Name        string `json:"name"`
	Target      string `json:"target"`
	Family      string `json:"family,omitempty"`
	Artwork     string `json:"artwork,omitempty"`
	Description string `json:"description,omitempty"`
}

type hardwareRequirement struct {
	Name     string `json:"name"`
	Docs     string `json:"docs,omitempty"`
	Optional bool   `json:"optional,omitempty"`
}

type standardCatalog struct {
	Packages  map[string]standardPackage `json:"packages"`
	Platforms map[string]standardPackage `json:"platforms,omitempty"`
	Libc      []string                   `json:"libc,omitempty"`
	Module    string                     `json:"module,omitempty"`
	Builtins  packageDocs                `json:"builtins"`
}

type customTarget struct {
	Name       string
	Label      string
	Definition string
	Backend    string
	Format     string
	RTGSource  string
	RTGImports []targetDefinitionAsset
	Tags       []string
	Device     string
	Hidden     bool
}

type boardExample struct {
	Path     string                `json:"path"`
	Language string                `json:"language,omitempty"`
	Hardware []hardwareRequirement `json:"hardware,omitempty"`
}

type boardDefinition struct {
	Name       string         `json:"name"`
	Target     string         `json:"target"`
	Tag        string         `json:"tag"`
	Docs       string         `json:"docs,omitempty"`
	Artwork    string         `json:"artwork,omitempty"`
	Machine    string         `json:"machine"`
	Definition string         `json:"definition"`
	Backend    string         `json:"backend"`
	Packages   []string       `json:"packages"`
	Examples   []boardExample `json:"examples"`
}

var customTargets = []customTarget{
	{Name: "esp32c6-jtag/riscv32", Definition: "backends/esp32c6_jtag.rtg", Backend: "backends/esp32c6-jtag-riscv32.wasm", Tags: []string{"m5nanoc6"}, Device: "esp32", Hidden: true},
	{Name: "msdos/8086", Label: "MS-DOS 8086 (.COM)", Definition: "backends/msdos.rtg", Backend: "backends/msdos-8086.rnvb", Format: "vm32", RTGSource: "backends/msdos.rtg", RTGImports: pc8086DefinitionImports(), Device: "computer"},
	{Name: "msdos/8086-mz", Label: "MS-DOS 8086 (.EXE)", Definition: "backends/msdos.rtg", Backend: "backends/msdos-8086-mz.rnvb", Format: "vm32", RTGSource: "backends/msdos.rtg", RTGImports: pc8086DefinitionImports(), Device: "computer"},
	{Name: "bios/8086", Label: "PC BIOS 8086 (boot disk)", Definition: "backends/msdos.rtg", Backend: "backends/bios-8086.rnvb", Format: "vm32", RTGSource: "backends/msdos.rtg", RTGImports: pc8086DefinitionImports(), Device: "computer"},
	{Name: "uefi/amd64", Label: "UEFI x86-64 (.EFI)", Definition: "backends/uefi_amd64.rtg", Backend: "backends/uefi-amd64.rnvb", Format: "vm32", RTGSource: "backends/uefi_amd64.rtg", RTGImports: []targetDefinitionAsset{
		{Name: "backend/definitions/x86_64.rtg", Source: "backends/definitions/x86_64.rtg"},
		{Name: "backend/definitions/elf_amd64_primitives.rtg", Source: "backends/definitions/elf_amd64_primitives.rtg"},
	}, Device: "computer"},
}

func pc8086DefinitionImports() []targetDefinitionAsset {
	return []targetDefinitionAsset{{Name: ".renvo/bios_8086.rtg", Source: "backends/bios_8086.rtg"}}
}

func main() {
	output := flag.String("o", "sandbox/wasm/browser", "browser asset output directory")
	flag.Parse()
	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	if err = os.MkdirAll(*output, 0o755); err != nil {
		fail(err)
	}
	boards, err := readBoardDefinitions(root)
	if err != nil {
		fail(err)
	}
	catalog := targetCatalog{
		Compiler: "renvo.wasm", Linker: "renvo-linker.wasm", LanguageService: "renvo-language-service.wasm", Formatter: "renvo-format.wasm",
		BackendJIT: "renvo-backend-jit.wasm", VMBackend: "renvo-vm-backend.wasm",
		BrowserPrefix: "browser-host-prefix.html", BrowserSuffix: "browser-host-suffix.html",
		Stdlib: "stdlib/catalog.json",
	}
	for _, descriptor := range targetinfo.All() {
		if !descriptor.Advertised {
			continue
		}
		backend := "backends/native.wasm"
		if descriptor.Backend == "wasi/wasm32" {
			backend = "backends/wasi-wasm32.wasm"
		}
		catalog.Targets = append(catalog.Targets, targetAsset{
			Name: descriptor.Name, Label: targetLabel(descriptor.Name), BackendTarget: descriptor.Backend,
			Backend: backend, CBackend: "backends/native-c.wasm", Output: outputName(descriptor.Name, descriptor.Image),
			Runnable: descriptor.Backend == "wasi/wasm32", Tags: descriptor.Tags,
		})
	}
	for _, custom := range customTargets {
		source, readErr := os.ReadFile(custom.Definition)
		if readErr != nil {
			fail(readErr)
		}
		resolved := backenddef.ResolveImports(source, custom.Definition, custom.Name, filesystemImports{})
		if !resolved.Ok {
			fail(fmt.Errorf("resolve %s: %s", custom.Definition, resolved.Message))
		}
		descriptor := resolved.Descriptor
		tags := append([]string(nil), descriptor.BuildTags...)
		tags = append(tags, custom.Tags...)
		catalog.Targets = append(catalog.Targets, targetAsset{
			Name: descriptor.Name, Label: custom.Label, BackendTarget: descriptor.Name, Backend: custom.Backend, BackendFormat: custom.Format,
			RTGDefinition: custom.RTGSource, RTGDefinitionName: packagedDefinitionName(custom.RTGSource), RTGImports: custom.RTGImports,
			Output: outputName(descriptor.Name, descriptor.OutputKind), Tags: tags,
			Definition: hex.EncodeToString(descriptor.Definition[:]), DescriptorVersion: descriptor.Version,
			Device: custom.Device,
			Hidden: custom.Hidden,
		})
	}
	for _, board := range boards {
		source, readErr := os.ReadFile(board.Definition)
		if readErr != nil {
			fail(readErr)
		}
		resolved := backenddef.ResolveImports(source, board.Definition, board.Machine, filesystemImports{})
		if !resolved.Ok {
			fail(fmt.Errorf("resolve %s: %s", board.Definition, resolved.Message))
		}
		descriptor := resolved.Descriptor
		tags := append([]string(nil), descriptor.BuildTags...)
		tags = append(tags, board.Tag)
		catalog.Targets = append(catalog.Targets, targetAsset{
			Name: board.Target, Label: board.Name, FrontendTarget: board.Machine, BackendTarget: board.Machine,
			Backend: board.Backend, Output: outputName(board.Machine, descriptor.OutputKind),
			Tags: tags, Definition: hex.EncodeToString(descriptor.Definition[:]),
			DescriptorVersion: descriptor.Version, Device: "esp32", Docs: board.Docs, Artwork: board.Artwork,
		})
	}
	for _, asset := range []*string{&catalog.Compiler, &catalog.Linker, &catalog.LanguageService, &catalog.Formatter, &catalog.BackendJIT, &catalog.VMBackend} {
		if *asset, err = versionAsset(*output, *asset); err != nil {
			fail(err)
		}
	}
	for i := range catalog.Targets {
		if catalog.Targets[i].Backend, err = versionAsset(*output, catalog.Targets[i].Backend); err != nil {
			fail(err)
		}
		if catalog.Targets[i].CBackend != "" {
			if catalog.Targets[i].CBackend, err = versionAsset(*output, catalog.Targets[i].CBackend); err != nil {
				fail(err)
			}
		}
		if catalog.Targets[i].RTGDefinition != "" {
			if catalog.Targets[i].RTGDefinition, err = versionAsset(*output, catalog.Targets[i].RTGDefinition); err != nil {
				fail(err)
			}
		}
		for j := range catalog.Targets[i].RTGImports {
			if catalog.Targets[i].RTGImports[j].Source, err = versionAsset(*output, catalog.Targets[i].RTGImports[j].Source); err != nil {
				fail(err)
			}
		}
	}
	if err = writeJSON(filepath.Join(*output, "targets.json"), catalog); err != nil {
		fail(err)
	}
	if err = writeBrowserHost(*output); err != nil {
		fail(err)
	}
	if err = buildStandardLibrary(root, *output, boards); err != nil {
		fail(err)
	}
}

func packagedDefinitionName(source string) string {
	return filepath.ToSlash(filepath.Join(".renvo", filepath.Base(source)))
}

func targetLabel(name string) string {
	switch name {
	case "linux/amd64":
		return "Linux x86-64"
	case "linux/386":
		return "Linux x86 (32-bit)"
	case "linux/aarch64":
		return "Linux ARM64"
	case "linux/arm":
		return "Linux ARM (32-bit)"
	case "windows/amd64":
		return "Windows x86-64"
	case "windows/386":
		return "Windows x86 (32-bit)"
	case "windows/arm64":
		return "Windows ARM64"
	case "darwin/arm64":
		return "macOS ARM64"
	case "wasi/wasm32":
		return "WebAssembly (WASI)"
	case "browser/wasm32":
		return "Web application"
	case "vm/vm32":
		return "Renvo VM bytecode"
	case "freebsd/amd64":
		return "FreeBSD x86-64"
	case "openbsd/amd64":
		return "OpenBSD x86-64"
	case "netbsd/amd64":
		return "NetBSD x86-64"
	}
	return name
}

func versionAsset(output, name string) (string, error) {
	if name == "" || strings.Contains(name, "://") {
		return name, nil
	}
	path := strings.SplitN(name, "?", 2)[0]
	data, err := os.ReadFile(filepath.Join(output, filepath.FromSlash(path)))
	if err != nil {
		return "", fmt.Errorf("version %s: %w", name, err)
	}
	sum := sha256.Sum256(data)
	return path + "?v=" + hex.EncodeToString(sum[:6]), nil
}

func readBoardDefinitions(root string) ([]boardDefinition, error) {
	source, err := os.ReadFile(filepath.Join(root, "device", "board", "catalog.json"))
	if err != nil {
		return nil, err
	}
	var boards []boardDefinition
	if err := json.Unmarshal(source, &boards); err != nil {
		return nil, err
	}
	seenTargets := make(map[string]bool)
	seenTags := make(map[string]bool)
	for _, board := range boards {
		if board.Name == "" || board.Target == "" || board.Tag == "" || board.Machine == "" ||
			board.Definition == "" || board.Backend == "" || board.Docs == "" || board.Artwork == "" {
			return nil, fmt.Errorf("incomplete board definition for %q", board.Target)
		}
		for _, example := range board.Examples {
			if example.Path == "" {
				return nil, fmt.Errorf("board %q has an example without a path", board.Target)
			}
			for _, hardware := range example.Hardware {
				if hardware.Name == "" || hardware.Docs == "" {
					return nil, fmt.Errorf("example %q has incomplete hardware metadata", example.Path)
				}
			}
		}
		if seenTargets[board.Target] || seenTags[board.Tag] {
			return nil, fmt.Errorf("duplicate board target or tag for %q", board.Target)
		}
		seenTargets[board.Target] = true
		seenTags[board.Tag] = true
	}
	return boards, nil
}

func writeBrowserHost(output string) error {
	const marker = `const wasm64="`
	packaged := driver.PackageBrowserHTML(nil)
	at := bytes.Index(packaged, []byte(marker))
	if at < 0 {
		return fmt.Errorf("browser host marker is missing")
	}
	prefixEnd := at + len(marker)
	if prefixEnd >= len(packaged) || packaged[prefixEnd] != '"' {
		return fmt.Errorf("browser host marker has unexpected contents")
	}
	if err := os.WriteFile(filepath.Join(output, "browser-host-prefix.html"), packaged[:prefixEnd], 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(output, "browser-host-suffix.html"), packaged[prefixEnd:], 0o644)
}

func outputName(target string, image string) string {
	if target == "wasi/wasm32" || target == "browser/wasm32" {
		return "app.wasm"
	}
	if strings.HasPrefix(target, "windows/") {
		return "app.exe"
	}
	if target == "vm/vm32" {
		return "app.rnvb"
	}
	if image == "dos-com" {
		return "app.com"
	}
	if image == "dos-mz" {
		return "app.exe"
	}
	if image == "uefi-pe" {
		return "BOOTX64.EFI"
	}
	if image == "bios-disk" {
		return "renvo-bios.img"
	}
	if strings.HasPrefix(target, "esp32") || strings.Contains(image, "elf") {
		return "app.elf"
	}
	return "app"
}

func buildStandardLibrary(root string, output string, boards []boardDefinition) error {
	stdRoot := filepath.Join(root, "std")
	packages := make(map[string]standardPackage)
	err := filepath.WalkDir(stdRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			return readErr
		}
		hasSource := false
		for _, child := range entries {
			if !child.IsDir() && strings.HasSuffix(child.Name(), ".go") && !strings.HasSuffix(child.Name(), "_test.go") {
				hasSource = true
				break
			}
		}
		if !hasSource {
			return nil
		}
		relative, relErr := filepath.Rel(stdRoot, path)
		if relErr != nil {
			return relErr
		}
		name := filepath.ToSlash(relative)
		docs, docsErr := buildPackageDocs(path, name)
		if docsErr != nil {
			return docsErr
		}
		item := standardPackage{Docs: docs}
		imports := make(map[string]bool)
		for _, child := range entries {
			if child.IsDir() || strings.HasSuffix(child.Name(), "_test.go") || strings.HasPrefix(child.Name(), ".") {
				continue
			}
			sourcePath := filepath.Join(path, child.Name())
			targetPath := filepath.Join(output, "stdlib", "src", relative, child.Name())
			if copyErr := copyFile(sourcePath, targetPath); copyErr != nil {
				return copyErr
			}
			item.Files = append(item.Files, child.Name())
			if strings.HasSuffix(child.Name(), ".go") {
				for _, imported := range sourceImports(sourcePath) {
					imports[imported] = true
				}
			}
		}
		for imported := range imports {
			item.Imports = append(item.Imports, imported)
		}
		sort.Strings(item.Files)
		sort.Strings(item.Imports)
		packages[name] = item
		return nil
	})
	if err != nil {
		return err
	}
	platforms, err := buildPlatformPackages(root, output, boards)
	if err != nil {
		return err
	}
	libc, err := buildCLibrary(root, output)
	if err != nil {
		return err
	}
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(output, "stdlib", "catalog.json"), standardCatalog{
		Packages: packages, Platforms: platforms, Libc: libc, Module: string(module), Builtins: builtinDocs(),
	})
}

func buildCLibrary(root string, output string) ([]string, error) {
	libcRoot := filepath.Join(root, "libc")
	var files []string
	err := filepath.WalkDir(libcRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			return nil
		}
		relative, err := filepath.Rel(libcRoot, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		files = append(files, name)
		return copyFile(path, filepath.Join(output, "stdlib", "libc", relative))
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

type platformPackageSpec struct {
	Path      string
	Target    string
	ArenaSize int
	Board     string
	Boards    []boardTarget
	Computers []computerTarget
	Language  string
}

func platformPackageSpecs(boards []boardDefinition) []platformPackageSpec {
	dosComputer := []computerTarget{{
		Name: "IBM PC compatible", Target: "msdos/8086-mz", Family: "Retro computer", Artwork: "ibmpc",
		Description: "IBM PC-compatible running MS-DOS, FreeDOS, or a compatible emulator",
	}}
	uefiComputer := []computerTarget{{
		Name: "x86-64 UEFI system", Target: "uefi/amd64", Family: "Firmware", Artwork: "ibmpc",
		Description: "PC or virtual machine with x86-64 UEFI firmware",
	}}
	biosComputer := []computerTarget{{
		Name: "IBM PC-compatible BIOS", Target: "bios/8086", Family: "Firmware", Artwork: "ibmpc",
		Description: "PC or virtual machine with legacy IBM PC-compatible BIOS firmware",
	}}
	specs := []platformPackageSpec{
		{Path: "forms"},
		{
			Path: "examples/pdp11v7", Target: "unixv7/pdp11",
			Computers: []computerTarget{{
				Name: "PDP-11", Target: "unixv7/pdp11", Family: "Retro computer", Artwork: "pdp11",
				Description: "PDP-11 running Unix V7, or a compatible emulator",
			}},
		},
		{
			Path: "examples/msdos", Target: "msdos/8086",
			Computers: []computerTarget{{
				Name: "IBM PC compatible", Target: "msdos/8086", Family: "Retro computer", Artwork: "ibmpc",
				Description: "IBM PC-compatible running MS-DOS, FreeDOS, or a compatible emulator",
			}},
		},
		{Path: "examples/msdos-vga", Target: "msdos/8086-mz", ArenaSize: 4096, Computers: dosComputer},
		{Path: "examples/msdos-filesystem", Target: "msdos/8086-mz", ArenaSize: 4096, Computers: dosComputer},
		{Path: "examples/msdos-system", Target: "msdos/8086-mz", ArenaSize: 4096, Computers: dosComputer},
		{Path: "examples/msdos-input", Target: "msdos/8086-mz", ArenaSize: 4096, Computers: dosComputer},
		{Path: "examples/bios-hello", Target: "bios/8086", Computers: biosComputer},
		{Path: "examples/uefi-hello", Target: "uefi/amd64", Computers: uefiComputer},
		{Path: "examples/uefi-graphics", Target: "uefi/amd64", Computers: uefiComputer},
		{Path: "examples/uefi-filesystem", Target: "uefi/amd64", Computers: uefiComputer},
		{Path: "examples/uefi-linux-boot", Target: "uefi/amd64", Computers: uefiComputer},
		{Path: "examples/c-sqlite-hash", Target: "linux/amd64", Language: "c"},
		{Path: "device/dos"},
		{Path: "device/bios"},
		{Path: "device/uefi"},
		{Path: "device/mmio"},
		{Path: "device/gpio"},
		{Path: "device/clock"},
		{Path: "device/i2c"},
		{Path: "device/uart"},
		{Path: "device/terminal"},
		{Path: "device/input/tca8418"},
		{Path: "device/input/st7121"},
		{Path: "device/display/st7121"},
		{Path: "device/sensor/sgp30"},
		{Path: "device/sensor/adxl345"},
		{Path: "device/sensor/bme688"},
		{Path: "device/sensor/miniscale"},
		{Path: "device/audio/sam2695"},
		{Path: "device/ws2812"},
		{Path: "device/internal/esprmt"},
		{Path: "internal/arena"},
	}
	// The public board package contains all build-tagged adapters and is loaded
	// for every board target. Wiring packages and examples come from the board
	// catalog so adding a board does not require another editor-side list.
	specs = append(specs, platformPackageSpec{Path: "device/board"})
	for _, board := range boards {
		for _, path := range board.Packages {
			specs = append(specs, platformPackageSpec{Path: path})
		}
		for _, example := range board.Examples {
			found := -1
			for i := range specs {
				if specs[i].Path == example.Path {
					found = i
					break
				}
			}
			if found < 0 {
				specs = append(specs, platformPackageSpec{Path: example.Path, Language: example.Language})
				found = len(specs) - 1
			}
			specs[found].Boards = append(specs[found].Boards, boardTarget{
				Name: board.Name, Target: board.Target, Docs: board.Docs, Artwork: board.Artwork, Hardware: example.Hardware,
			})
		}
	}
	return specs
}

func buildPlatformPackages(root string, output string, boards []boardDefinition) (map[string]standardPackage, error) {
	packages := make(map[string]standardPackage)
	for _, spec := range platformPackageSpecs(boards) {
		path := filepath.Join(root, filepath.FromSlash(spec.Path))
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		item := standardPackage{
			Root: spec.Path, Target: spec.Target, ArenaSize: spec.ArenaSize, Board: spec.Board, Boards: spec.Boards,
			Computers: spec.Computers, Language: spec.Language,
		}
		imports := make(map[string]bool)
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			if entry.IsDir() {
				err = filepath.WalkDir(filepath.Join(path, entry.Name()), func(assetPath string, asset os.DirEntry, walkErr error) error {
					if walkErr != nil {
						return walkErr
					}
					if asset.IsDir() || strings.HasPrefix(asset.Name(), ".") || strings.HasSuffix(asset.Name(), "_test.go") {
						return nil
					}
					relative, relErr := filepath.Rel(path, assetPath)
					if relErr != nil {
						return relErr
					}
					item.Files = append(item.Files, filepath.ToSlash(relative))
					return copyFile(assetPath, filepath.Join(output, "stdlib", "module", filepath.FromSlash(spec.Path), relative))
				})
				if err != nil {
					return nil, err
				}
				continue
			}
			if strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			source := filepath.Join(path, entry.Name())
			destination := filepath.Join(output, "stdlib", "module", filepath.FromSlash(spec.Path), entry.Name())
			if err = copyFile(source, destination); err != nil {
				return nil, err
			}
			// Retain package licenses in the deployable bundle without presenting
			// them to the compiler as source or embeddable package data.
			if entry.Name() == "LICENSE" {
				continue
			}
			item.Files = append(item.Files, entry.Name())
			if strings.HasSuffix(entry.Name(), ".go") {
				for _, imported := range sourceImports(source) {
					imports[imported] = true
				}
				if sourcePackage(source) == "main" {
					item.Main = true
				}
			}
		}
		for imported := range imports {
			item.Imports = append(item.Imports, imported)
		}
		sort.Strings(item.Files)
		sort.Strings(item.Imports)
		importPath := "renvo.dev/" + spec.Path
		if !item.Main {
			item.Docs, err = buildPackageDocs(path, importPath)
			if err != nil {
				return nil, err
			}
		}
		packages[importPath] = item
	}
	return packages, nil
}

func buildPackageDocs(dir string, importPath string) (*packageDocs, error) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("document %s: %w", importPath, err)
	}
	files := make([]*ast.File, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return nil, fmt.Errorf("document %s: %w", importPath, parseErr)
		}
		files = append(files, file)
	}
	if len(files) == 0 {
		return nil, nil
	}
	pkg, err := doc.NewFromFiles(fset, files, importPath, doc.PreserveAST)
	if err != nil {
		return nil, fmt.Errorf("document %s: %w", importPath, err)
	}
	result := &packageDocs{Name: pkg.Name, ImportPath: importPath, Doc: cleanPackageDoc(pkg.Name, pkg.Doc), Examples: formatDocExamples(fset, pkg.Examples)}
	appendValues := func(destination *[]docEntry, values []*doc.Value) error {
		for _, value := range values {
			signature, formatErr := formatGenDeclSignature(fset, value.Decl)
			if formatErr != nil {
				return formatErr
			}
			file, line := docSource(fset, value.Decl)
			*destination = append(*destination, docEntry{Name: strings.Join(value.Names, ", "), Signature: signature,
				Doc: strings.TrimSpace(value.Doc), File: file, Line: line})
		}
		return nil
	}
	if err = appendValues(&result.Constants, pkg.Consts); err != nil {
		return nil, fmt.Errorf("document constants in %s: %w", importPath, err)
	}
	if err = appendValues(&result.Variables, pkg.Vars); err != nil {
		return nil, fmt.Errorf("document variables in %s: %w", importPath, err)
	}
	appendFunction := func(function *doc.Func) error {
		signature, formatErr := formatFunctionSignature(fset, function.Decl)
		if formatErr != nil {
			return fmt.Errorf("document function %s.%s: %w", importPath, function.Name, formatErr)
		}
		file, line := docSource(fset, function.Decl)
		result.Functions = append(result.Functions, docEntry{Name: function.Name, Signature: signature, Doc: strings.TrimSpace(function.Doc), File: file, Line: line,
			Examples: formatDocExamples(fset, function.Examples)})
		return nil
	}
	for _, function := range pkg.Funcs {
		if err = appendFunction(function); err != nil {
			return nil, err
		}
	}
	for _, typ := range pkg.Types {
		for _, function := range typ.Funcs {
			if err = appendFunction(function); err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(result.Functions, func(i, j int) bool { return result.Functions[i].Name < result.Functions[j].Name })
	for _, typ := range pkg.Types {
		signature, formatErr := formatGenDeclSignature(fset, typ.Decl)
		if formatErr != nil {
			return nil, fmt.Errorf("document type %s.%s: %w", importPath, typ.Name, formatErr)
		}
		file, line := docSource(fset, typ.Decl)
		entry := docType{docEntry: docEntry{Name: typ.Name, Signature: signature, Doc: strings.TrimSpace(typ.Doc), File: file, Line: line,
			Examples: formatDocExamples(fset, typ.Examples)}}
		for _, method := range typ.Methods {
			methodSignature, methodErr := formatFunctionSignature(fset, method.Decl)
			if methodErr != nil {
				return nil, fmt.Errorf("document method %s.%s: %w", importPath, method.Name, methodErr)
			}
			methodFile, methodLine := docSource(fset, method.Decl)
			entry.Methods = append(entry.Methods, docEntry{Name: method.Name, Signature: methodSignature, Doc: strings.TrimSpace(method.Doc), File: methodFile, Line: methodLine,
				Examples: formatDocExamples(fset, method.Examples)})
		}
		result.Types = append(result.Types, entry)
	}
	return result, nil
}

func formatDocExamples(fset *token.FileSet, examples []*doc.Example) []docExample {
	result := make([]docExample, 0, len(examples))
	for _, example := range examples {
		comments := make([]*ast.CommentGroup, 0, len(example.Comments))
		for _, group := range example.Comments {
			text := strings.ToLower(strings.TrimSpace(group.Text()))
			if strings.HasPrefix(text, "output:") || strings.HasPrefix(text, "unordered output:") {
				continue
			}
			comments = append(comments, group)
		}
		var output bytes.Buffer
		node := &printer.CommentedNode{Node: example.Code, Comments: comments}
		if err := format.Node(&output, fset, node); err != nil {
			continue
		}
		code := strings.TrimSpace(output.String())
		if _, block := example.Code.(*ast.BlockStmt); block {
			code = strings.TrimPrefix(code, "{")
			code = strings.TrimSuffix(code, "}")
			code = strings.TrimSpace(code)
			lines := strings.Split(code, "\n")
			for i := range lines {
				lines[i] = strings.TrimPrefix(lines[i], "\t")
			}
			code = strings.Join(lines, "\n")
		}
		if code == "" {
			continue
		}
		result = append(result, docExample{Name: example.Suffix, Doc: strings.TrimSpace(example.Doc), Code: code,
			Output: strings.TrimSpace(example.Output), Unordered: example.Unordered})
	}
	return result
}

func docSource(fset *token.FileSet, node ast.Node) (string, int) {
	position := fset.Position(node.Pos())
	return filepath.Base(position.Filename), position.Line
}

func cleanPackageDoc(name string, text string) string {
	paragraphs := strings.Split(strings.TrimSpace(text), "\n\n")
	seen := make(map[string]bool)
	result := make([]string, 0, len(paragraphs))
	packageIntroduction := false
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" || seen[paragraph] {
			continue
		}
		introduction := strings.HasPrefix(paragraph, "Package "+name+" ")
		if introduction && packageIntroduction {
			continue
		}
		seen[paragraph] = true
		packageIntroduction = packageIntroduction || introduction
		result = append(result, paragraph)
	}
	return strings.Join(result, "\n\n")
}

func formatFunctionSignature(fset *token.FileSet, declaration *ast.FuncDecl) (string, error) {
	copy := *declaration
	copy.Doc = nil
	copy.Body = nil
	return formatDocNode(fset, &copy)
}

func formatGenDeclSignature(fset *token.FileSet, declaration *ast.GenDecl) (string, error) {
	copy := *declaration
	copy.Doc = nil
	copy.Specs = make([]ast.Spec, 0, len(declaration.Specs))
	for _, spec := range declaration.Specs {
		switch value := spec.(type) {
		case *ast.ValueSpec:
			clean := *value
			clean.Doc = nil
			clean.Comment = nil
			copy.Specs = append(copy.Specs, &clean)
		case *ast.TypeSpec:
			clean := *value
			clean.Doc = nil
			clean.Comment = nil
			copy.Specs = append(copy.Specs, &clean)
		}
	}
	return formatDocNode(fset, &copy)
}

func formatDocNode(fset *token.FileSet, node ast.Node) (string, error) {
	var output bytes.Buffer
	if err := format.Node(&output, fset, node); err != nil {
		return "", err
	}
	return strings.TrimSpace(output.String()), nil
}

func builtinDocs() packageDocs {
	functions := []docEntry{
		{Name: "append", Signature: "func append(slice []T, values ...T) []T", Doc: "Appends values to a slice and returns the resulting slice."},
		{Name: "cap", Signature: "func cap(value T) int", Doc: "Returns the capacity of an array, slice, or channel."},
		{Name: "clear", Signature: "func clear(value T)", Doc: "Deletes all map entries or zeroes all slice elements."},
		{Name: "close", Signature: "func close(channel chan<- T)", Doc: "Closes a channel. A receive can continue until buffered values have been received."},
		{Name: "complex", Signature: "func complex(real, imag T) ComplexT", Doc: "Constructs a complex value from real and imaginary components."},
		{Name: "copy", Signature: "func copy(dst, src []T) int", Doc: "Copies elements and returns the number copied. Source and destination may overlap."},
		{Name: "delete", Signature: "func delete(m map[K]V, key K)", Doc: "Deletes the map entry for key. Deleting from a nil map or deleting a missing key has no effect."},
		{Name: "imag", Signature: "func imag(value ComplexT) T", Doc: "Returns the imaginary component of a complex value."},
		{Name: "len", Signature: "func len(value T) int", Doc: "Returns the length of a string, array, slice, map, or channel."},
		{Name: "make", Signature: "func make(T, size ...int) T", Doc: "Allocates and initializes a slice, map, or channel."},
		{Name: "max", Signature: "func max(values ...T) T", Doc: "Returns the largest value from one or more ordered values."},
		{Name: "min", Signature: "func min(values ...T) T", Doc: "Returns the smallest value from one or more ordered values."},
		{Name: "new", Signature: "func new(T) *T", Doc: "Allocates a zero value and returns a pointer to it."},
		{Name: "panic", Signature: "func panic(value any)", Doc: "Stops normal execution and begins panicking. Deferred functions run while the panic moves up the call stack."},
		{Name: "print", Signature: "func print(values ...T)", Doc: "Writes values using Renvo's implementation-defined debug format."},
		{Name: "println", Signature: "func println(values ...T)", Doc: "Writes values using Renvo's implementation-defined debug format followed by a newline."},
		{Name: "real", Signature: "func real(value ComplexT) T", Doc: "Returns the real component of a complex value."},
		{Name: "recover", Signature: "func recover() any", Doc: "Stops a panicking sequence when called directly by a deferred function and returns the value passed to panic."},
	}
	typeNames := []string{"any", "bool", "byte", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr"}
	types := make([]docType, 0, len(typeNames))
	for _, name := range typeNames {
		description := "Predeclared type."
		switch name {
		case "any":
			description = "Predeclared alias for interface{}; it accepts a value of any type."
		case "byte":
			description = "Predeclared alias for uint8."
		case "rune":
			description = "Predeclared alias for int32, conventionally used for Unicode code points."
		case "error":
			description = "Predeclared interface for values that describe an error condition."
		}
		types = append(types, docType{docEntry: docEntry{Name: name, Signature: "type " + name, Doc: description}})
	}
	return packageDocs{
		Name: "builtin", ImportPath: "builtin",
		Doc: "Package builtin documents Renvo's predeclared identifiers. These names are available without an import.",
		Constants: []docEntry{
			{Name: "true, false", Signature: "const (\n\ttrue = 0 == 0\n\tfalse = 0 != 0\n)", Doc: "true and false are the two untyped boolean values."},
			{Name: "iota", Signature: "const iota = 0", Doc: "iota is the zero-based ordinal of the current const specification."},
		},
		Variables: []docEntry{{Name: "nil", Signature: "var nil T", Doc: "nil is the zero value for pointer, channel, function, interface, map, and slice types."}},
		Functions: functions, Types: types,
	}
}

func sourcePackage(path string) string {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.PackageClauseOnly)
	if err != nil || file.Name == nil {
		return ""
	}
	return file.Name.Name
}

func sourceImports(path string) []string {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return nil
	}
	var imports []string
	for _, spec := range file.Imports {
		value, unquoteErr := strconv.Unquote(spec.Path.Value)
		if unquoteErr == nil && value != "C" {
			imports = append(imports, value)
		}
	}
	return imports
}

func copyFile(source string, target string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

type filesystemImports struct{}

func (filesystemImports) LoadImport(importingFilename string, importPath string) rtg.ImportSource {
	path := filepath.Clean(filepath.Join(filepath.Dir(importingFilename), importPath))
	source, err := os.ReadFile(path)
	return rtg.ImportSource{Source: source, Filename: path, Ok: err == nil}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "browserassets:", err)
	os.Exit(1)
}
