package main

import (
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
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/targetinfo"
)

type targetAsset struct {
	Name              string   `json:"name"`
	BackendTarget     string   `json:"backendTarget"`
	Backend           string   `json:"backend"`
	Output            string   `json:"output"`
	Runnable          bool     `json:"runnable,omitempty"`
	Device            string   `json:"device,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Definition        string   `json:"definition,omitempty"`
	DescriptorVersion int      `json:"descriptorVersion,omitempty"`
}

type targetCatalog struct {
	LanguageService string        `json:"languageService"`
	Stdlib          string        `json:"stdlib"`
	Targets         []targetAsset `json:"targets"`
}

type standardPackage struct {
	Files   []string `json:"files"`
	Imports []string `json:"imports,omitempty"`
	Root    string   `json:"root,omitempty"`
	Main    bool     `json:"main,omitempty"`
	Target  string   `json:"target,omitempty"`
	Board   string   `json:"board,omitempty"`
}

type standardCatalog struct {
	Packages  map[string]standardPackage `json:"packages"`
	Platforms map[string]standardPackage `json:"platforms,omitempty"`
}

type customTarget struct {
	Name       string
	Definition string
	Backend    string
}

var customTargets = []customTarget{
	{Name: "esp32c6/riscv32", Definition: "backends/esp32c6.rtg", Backend: "backends/esp32c6-riscv32.wasm"},
	{Name: "esp32s3/xtensa_lx7", Definition: "backends/esp32s3.rtg", Backend: "backends/esp32s3-xtensa_lx7.wasm"},
	{Name: "esp32p4/riscv32", Definition: "backends/esp32p4.rtg", Backend: "backends/esp32p4-riscv32.wasm"},
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
	catalog := targetCatalog{LanguageService: "renvo-language-service.wasm", Stdlib: "stdlib/catalog.json"}
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
			Backend: backend, Output: outputName(descriptor.Name, descriptor.Image),
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
		catalog.Targets = append(catalog.Targets, targetAsset{
			Name: descriptor.Name, BackendTarget: descriptor.Name, Backend: custom.Backend,
			Output: outputName(descriptor.Name, descriptor.OutputKind), Tags: descriptor.BuildTags,
			Definition: hex.EncodeToString(descriptor.Definition[:]), DescriptorVersion: descriptor.Version,
			Device: "esp32",
		})
	}
	if err = writeJSON(filepath.Join(*output, "targets.json"), catalog); err != nil {
		fail(err)
	}
	if err = buildStandardLibrary(root, *output); err != nil {
		fail(err)
	}
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
	if strings.HasPrefix(target, "esp32") || strings.Contains(image, "elf") {
		return "app.elf"
	}
	return "app"
}

func buildStandardLibrary(root string, output string) error {
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
	platforms, err := buildPlatformPackages(root, output)
	if err != nil {
		return err
	}
	return writeJSON(filepath.Join(output, "stdlib", "catalog.json"), standardCatalog{Packages: packages, Platforms: platforms})
}

type platformPackageSpec struct {
	Path   string
	Target string
	Board  string
}

func platformPackageSpecs() []platformPackageSpec {
	return []platformPackageSpec{
		{Path: "forms"},
		{Path: "device/mmio"},
		{Path: "device/gpio"},
		{Path: "device/clock"},
		{Path: "device/i2c"},
		{Path: "device/terminal"},
		{Path: "device/input/tca8418"},
		{Path: "device/input/st7121"},
		{Path: "device/display/st7121"},
		{Path: "device/sensor/sgp30"},
		{Path: "device/sensor/adxl345"},
		{Path: "device/ws2812"},
		{Path: "device/internal/esprmt"},
		{Path: "device/esp32c6", Target: "esp32c6/riscv32"},
		{Path: "device/board/m5nanoc6", Target: "esp32c6/riscv32"},
		{Path: "examples/m5nanoc6/blink", Target: "esp32c6/riscv32", Board: "M5Stack NanoC6"},
		{Path: "examples/m5nanoc6/button_rgb", Target: "esp32c6/riscv32", Board: "M5Stack NanoC6"},
		{Path: "examples/m5nanoc6/air_quality", Target: "esp32c6/riscv32", Board: "M5Stack NanoC6"},
		{Path: "device/esp32s3", Target: "esp32s3/xtensa_lx7"},
		{Path: "device/board/m5atoms3lite", Target: "esp32s3/xtensa_lx7"},
		{Path: "examples/m5atoms3lite/adxl345", Target: "esp32s3/xtensa_lx7", Board: "M5Stack AtomS3 Lite"},
		{Path: "examples/m5atoms3lite/button_rgb", Target: "esp32s3/xtensa_lx7", Board: "M5Stack AtomS3 Lite"},
		{Path: "examples/m5atoms3lite/sk6812_strip", Target: "esp32s3/xtensa_lx7", Board: "M5Stack AtomS3 Lite"},
		{Path: "device/board/m5sticks3", Target: "esp32s3/xtensa_lx7"},
		{Path: "examples/m5sticks3/forms_menu", Target: "esp32s3/xtensa_lx7", Board: "M5Stack StickS3"},
		{Path: "device/board/m5cardputeradv", Target: "esp32s3/xtensa_lx7"},
		{Path: "examples/m5cardputeradv/terminal", Target: "esp32s3/xtensa_lx7", Board: "M5Stack Cardputer Adv"},
		{Path: "device/esp32p4", Target: "esp32p4/riscv32"},
		{Path: "device/board/m5tab5", Target: "esp32p4/riscv32"},
		{Path: "examples/m5tab5/fontcache", Target: "esp32p4/riscv32"},
		{Path: "examples/m5tab5/forms_demo", Target: "esp32p4/riscv32", Board: "M5Stack Tab5"},
		{Path: "examples/m5tab5/sgp30_demo", Target: "esp32p4/riscv32", Board: "M5Stack Tab5"},
		{Path: "examples/m5tab5/terminal", Target: "esp32p4/riscv32", Board: "M5Stack Tab5"},
		{Path: "examples/m5tab5/terminal_stress", Target: "esp32p4/riscv32", Board: "M5Stack Tab5"},
		{Path: "examples/m5tab5/touch_trails", Target: "esp32p4/riscv32", Board: "M5Stack Tab5"},
	}
}

func buildPlatformPackages(root string, output string) (map[string]standardPackage, error) {
	packages := make(map[string]standardPackage)
	for _, spec := range platformPackageSpecs() {
		path := filepath.Join(root, filepath.FromSlash(spec.Path))
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		item := standardPackage{Root: spec.Path, Target: spec.Target, Board: spec.Board}
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
