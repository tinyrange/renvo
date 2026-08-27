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
	found := false
	for _, declaration := range docs.Types {
		if declaration.Name == "Bus" {
			if declaration.File == "" || declaration.Line < 1 {
				t.Fatalf("Bus source location = %q:%d", declaration.File, declaration.Line)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("i2c docs do not contain exported Bus type: %#v", docs.Types)
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
