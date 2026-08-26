package main

import (
	"path/filepath"
	"strings"
	"testing"

	"renvo.dev/internal/targetinfo"
)

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
