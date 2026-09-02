package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/backenddef"
	"renvo.dev/internal/backendjit"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/targetinfo"
	"renvo.dev/internal/unit"
)

type browserDefinitionImports map[string][]byte

func (imports browserDefinitionImports) LoadImport(importingFilename, importPath string) rtg.ImportSource {
	name := filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(importingFilename), importPath)))
	source, ok := imports[name]
	return rtg.ImportSource{Source: source, Filename: name, Ok: ok}
}

func readBrowserDefinitionImport(root string, imported targetDefinitionAsset) ([]byte, error) {
	source, err := os.ReadFile(filepath.Join(root, imported.Source))
	if err == nil || strings.HasPrefix(imported.Name, ".renvo/") {
		return source, err
	}
	return os.ReadFile(filepath.Join(root, imported.Name))
}

func TestAdvertisedTargetsHaveHumanLabels(t *testing.T) {
	for _, descriptor := range targetinfo.All() {
		if !descriptor.Advertised {
			continue
		}
		label := targetLabel(descriptor.Name)
		if label == "" || label == descriptor.Name {
			t.Errorf("target %q has no human label", descriptor.Name)
		}
	}
}

func TestEmptyPackagedDefinitionHasNoName(t *testing.T) {
	if got := packagedDefinitionName(""); got != "" {
		t.Fatalf("packagedDefinitionName(empty) = %q, want empty", got)
	}
}

func TestBrowserCustomDefinitionsResolveFromPackagedNames(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	for _, target := range customTargets {
		if target.RTGSource == "" {
			continue
		}
		t.Run(target.Name, func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join(root, target.RTGSource))
			if err != nil {
				t.Fatal(err)
			}
			imports := browserDefinitionImports{}
			for _, imported := range target.RTGImports {
				imports[imported.Name], err = readBrowserDefinitionImport(root, imported)
				if err != nil {
					t.Fatal(err)
				}
			}
			resolved := backenddef.ResolveImports(source, packagedDefinitionName(target.RTGSource), target.Name, imports)
			if !resolved.Ok {
				t.Errorf("browser target cannot resolve its packaged definition: %s", resolved.Message)
			}
		})
	}
}

func TestBrowserUEFILinuxBootAssemblyEvaluates(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var target customTarget
	for _, candidate := range customTargets {
		if candidate.Name == "uefi/amd64" {
			target = candidate
			break
		}
	}
	if target.RTGSource == "" {
		t.Fatal("browser catalog does not publish uefi/amd64")
	}
	source, err := os.ReadFile(filepath.Join(root, target.RTGSource))
	if err != nil {
		t.Fatal(err)
	}
	imports := browserDefinitionImports{}
	for _, imported := range target.RTGImports {
		imports[imported.Name], err = readBrowserDefinitionImport(root, imported)
		if err != nil {
			t.Fatal(err)
		}
	}
	resolved := rtg.ResolveDefinitions(rtg.ParseImports(source, packagedDefinitionName(target.RTGSource), imports))
	if !resolved.Ok || len(resolved.Targets) != 1 {
		t.Fatalf("resolve browser UEFI definition: %#v", resolved.Diagnostics)
	}
	descriptor := resolved.Targets[0].Descriptor
	project := filepath.Join(root, "examples", "uefi-linux-boot")
	stdRoot := filepath.Join(root, "std")
	built := driver.BuildPackageUnitCompact(".", descriptor.Name, descriptor.BuildTags, project, stdRoot, driver.OSFS{})
	if !built.Ok {
		t.Fatalf("build browser compact unit: %#v", built.Diagnostic)
	}
	bound, ok := unit.BindTarget(built.Unit, unit.TargetBinding{
		Target: descriptor.Name, Definition: string(descriptor.Definition[:]), DescriptorVersion: descriptor.Version,
	})
	if !ok {
		t.Fatal("bind browser compact unit to UEFI target")
	}
	_, evaluated := backendjit.EvaluateRTGAssembly(bound, resolved, descriptor, stdRoot, backendcompiled.Backend{})
	if !evaluated.Ok {
		t.Fatalf("evaluate browser UEFI RTGASM: %#v", evaluated.Diagnostic)
	}
}

func TestPackageDocumentationIncludesOverviewAndDeclarations(t *testing.T) {
	docs, err := buildPackageDocs(filepath.Join("..", "..", "..", "..", "device", "i2c"), "renvo.dev/device/i2c")
	if err != nil {
		t.Fatal(err)
	}
	if docs == nil || docs.Name != "i2c" || docs.Doc == "" {
		t.Fatalf("package docs = %#v", docs)
	}
	constructor := false
	for _, function := range docs.Functions {
		if function.Name == "New" {
			constructor = true
			break
		}
	}
	if !constructor {
		t.Fatalf("i2c docs omit New constructor: %#v", docs.Functions)
	}
	found := false
	for _, declaration := range docs.Types {
		if declaration.Name == "Bus" {
			if declaration.File == "" || declaration.Line < 1 {
				t.Fatalf("Bus source location = %q:%d", declaration.File, declaration.Line)
			}
			for _, method := range declaration.Methods {
				if method.Name != "Device" {
					continue
				}
				if len(method.Examples) != 1 || !strings.Contains(method.Examples[0].Code, "i2c.New(board.Grove)") ||
					!strings.Contains(method.Examples[0].Code, "bus.Device(0x53)") || strings.Contains(method.Examples[0].Code, "fake") {
					t.Fatalf("Bus.Device examples = %#v", method.Examples)
				}
				found = true
			}
			break
		}
	}
	if !found {
		t.Fatalf("i2c docs do not contain exported Bus type: %#v", docs.Types)
	}
}

func TestPackageDocumentationIncludesTypedConstants(t *testing.T) {
	docs, err := buildPackageDocs(filepath.Join("..", "..", "..", "..", "std", "os"), "os")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, declaration := range docs.Constants {
		if declaration.Name == "ModePerm" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("os docs omit typed ModePerm constant: %#v", docs.Constants)
	}
}

func TestExternalDocumentationExamplesUseConsumerFacingAPIs(t *testing.T) {
	docs, err := buildPackageDocs(filepath.Join("..", "..", "..", "..", "device", "dos"), "renvo.dev/device/dos")
	if err != nil {
		t.Fatal(err)
	}
	examples := make([]docExample, 0)
	for _, function := range docs.Functions {
		examples = append(examples, function.Examples...)
	}
	for _, typ := range docs.Types {
		examples = append(examples, typ.Examples...)
		for _, method := range typ.Methods {
			examples = append(examples, method.Examples...)
		}
	}
	if len(examples) < 5 {
		t.Fatalf("DOS documentation only publishes %d examples", len(examples))
	}
	foundNow := false
	for _, example := range examples {
		if strings.Contains(example.Code, "dos.Now()") {
			foundNow = true
		}
		if strings.Contains(example.Code, "Hook") || strings.Contains(example.Code, "resetHooks") {
			t.Fatalf("test scaffolding leaked into documentation example: %s", example.Code)
		}
	}
	if !foundNow {
		t.Fatalf("DOS Now example is not presented as consumer code: %#v", examples)
	}
}

func TestI2CDriverDocumentationCoversTransactionLayers(t *testing.T) {
	docs, err := buildPackageDocs(filepath.Join("..", "..", "..", "..", "device", "i2c"), "renvo.dev/device/i2c")
	if err != nil {
		t.Fatal(err)
	}
	covered := make(map[string]bool)
	total := 0
	for _, function := range docs.Functions {
		if len(function.Examples) != 0 {
			covered[function.Name] = true
			total += len(function.Examples)
		}
	}
	for _, typ := range docs.Types {
		for _, method := range typ.Methods {
			if len(method.Examples) != 0 {
				covered[typ.Name+"."+method.Name] = true
				total += len(method.Examples)
			}
		}
	}
	for _, operation := range []string{"DefinePort", "NewBitBang", "BitBang.Tx", "Bus.Tx", "Device.Read", "Device.ReadAt", "Device.Write", "Device.WriteAt", "OperationError.Unwrap"} {
		if !covered[operation] {
			t.Errorf("I2C documentation has no %s example", operation)
		}
	}
	if total < 10 {
		t.Errorf("I2C documentation publishes %d driver examples, want at least 10", total)
	}
}

func TestHostOnlyHooksAreMarkedUnavailableToTargetPrograms(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "device")
	packages := map[string][]string{
		"dos":  {"InterruptHook", "PortInHook", "PortOutHook"},
		"bios": {"InterruptHook", "PortInHook", "PortOutHook", "BootDriveHook"},
		"uefi": {"ImageHandleHook", "SystemTableHook", "CallHook"},
	}
	for name, hooks := range packages {
		t.Run(name, func(t *testing.T) {
			docs, err := buildPackageDocs(filepath.Join(root, name), "renvo.dev/device/"+name)
			if err != nil {
				t.Fatal(err)
			}
			variables := make(map[string]docEntry, len(docs.Variables))
			for _, variable := range docs.Variables {
				variables[variable.Name] = variable
			}
			for _, hook := range hooks {
				variable, ok := variables[hook]
				if !ok || !strings.Contains(variable.Doc, "host builds") || !strings.Contains(variable.Doc, "unavailable") {
					t.Errorf("%s documentation does not identify its host-only scope: %#v", hook, variable)
				}
			}
		})
	}
}

func TestBuiltinDocumentationDescribesRenvoBuiltins(t *testing.T) {
	docs := builtinDocs()
	if docs.ImportPath != "builtin" || len(docs.Functions) < 10 || len(docs.Types) < 10 {
		t.Fatalf("builtin docs are incomplete: %#v", docs)
	}
	if docs.Functions[0].Name != "append" || docs.Functions[0].Doc == "" {
		t.Fatalf("append docs = %#v", docs.Functions[0])
	}
}

func TestPackageDocumentationKeepsOneBuildVariantOverview(t *testing.T) {
	doc := cleanPackageDoc("fmt", "Package fmt is portable.\n\nPackage fmt is target specific.\n\nMore detail.")
	if strings.Contains(doc, "target specific") || !strings.Contains(doc, "More detail.") {
		t.Fatalf("cleaned package doc = %q", doc)
	}
}

func TestEveryPublicCustomBrowserPackagePublishesAnExample(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	boards, err := readBoardDefinitions(root)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	total := 0
	for _, spec := range platformPackageSpecs(boards) {
		if seen[spec.Path] || strings.HasPrefix(spec.Path, "examples/") || strings.Contains(spec.Path, "/internal/") ||
			(spec.Path != "forms" && !strings.HasPrefix(spec.Path, "device/")) {
			continue
		}
		seen[spec.Path] = true
		docs, docsErr := buildPackageDocs(filepath.Join(root, filepath.FromSlash(spec.Path)), "renvo.dev/"+spec.Path)
		if docsErr != nil {
			t.Errorf("%s: %v", spec.Path, docsErr)
			continue
		}
		examples := append([]docExample(nil), docs.Examples...)
		for _, entry := range docs.Functions {
			examples = append(examples, entry.Examples...)
		}
		for _, entry := range docs.Types {
			examples = append(examples, entry.Examples...)
			for _, method := range entry.Methods {
				examples = append(examples, method.Examples...)
			}
		}
		if len(examples) == 0 {
			t.Errorf("%s has no documentation example", spec.Path)
		}
		for _, example := range examples {
			for _, scaffold := range []string{"fake", "recording", "resetHooks", "InterruptHook", "PortInHook", "PortOutHook"} {
				if strings.Contains(example.Code, scaffold) {
					t.Errorf("%s example contains test scaffolding %q: %s", spec.Path, scaffold, example.Code)
				}
			}
		}
		total += len(examples)
	}
	if total < 150 {
		t.Errorf("custom API catalog publishes %d examples, want at least 150", total)
	}
}
