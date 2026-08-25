package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/parser"
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
	Name              string   `json:"name"`
	Label             string   `json:"label,omitempty"`
	FrontendTarget    string   `json:"frontendTarget,omitempty"`
	BackendTarget     string   `json:"backendTarget"`
	Backend           string   `json:"backend"`
	CBackend          string   `json:"cBackend,omitempty"`
	Output            string   `json:"output"`
	Runnable          bool     `json:"runnable,omitempty"`
	Device            string   `json:"device,omitempty"`
	Docs              string   `json:"docs,omitempty"`
	Artwork           string   `json:"artwork,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Definition        string   `json:"definition,omitempty"`
	DescriptorVersion int      `json:"descriptorVersion,omitempty"`
	Hidden            bool     `json:"hidden,omitempty"`
}

type targetCatalog struct {
	Compiler        string        `json:"compiler"`
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
	Board     string           `json:"board,omitempty"`
	Boards    []boardTarget    `json:"boards,omitempty"`
	Computers []computerTarget `json:"computers,omitempty"`
	Language  string           `json:"language,omitempty"`
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
}

type customTarget struct {
	Name       string
	Definition string
	Backend    string
	Tags       []string
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
	{Name: "esp32c6-jtag/riscv32", Definition: "backends/esp32c6_jtag.rtg", Backend: "backends/esp32c6-jtag-riscv32.wasm", Tags: []string{"m5nanoc6"}, Hidden: true},
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
		Compiler: "renvo.wasm", LanguageService: "renvo-language-service.wasm", Formatter: "renvo-format.wasm",
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
			Name: descriptor.Name, BackendTarget: descriptor.Backend,
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
			Name: descriptor.Name, BackendTarget: descriptor.Name, Backend: custom.Backend,
			Output: outputName(descriptor.Name, descriptor.OutputKind), Tags: tags,
			Definition: hex.EncodeToString(descriptor.Definition[:]), DescriptorVersion: descriptor.Version,
			Device: "esp32",
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
	for _, asset := range []*string{&catalog.Compiler, &catalog.LanguageService, &catalog.Formatter, &catalog.BackendJIT, &catalog.VMBackend} {
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
		item := standardPackage{}
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
	return writeJSON(filepath.Join(output, "stdlib", "catalog.json"), standardCatalog{Packages: packages, Platforms: platforms, Libc: libc, Module: string(module)})
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
	Board     string
	Boards    []boardTarget
	Computers []computerTarget
	Language  string
}

func platformPackageSpecs(boards []boardDefinition) []platformPackageSpec {
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
				Description: "IBM PC-compatible running MS-DOS, or a compatible emulator",
			}},
		},
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
			Root: spec.Path, Target: spec.Target, Board: spec.Board, Boards: spec.Boards,
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
		packages["renvo.dev/"+spec.Path] = item
	}
	return packages, nil
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
