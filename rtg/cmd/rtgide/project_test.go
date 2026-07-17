package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedHelloFormIsGoSourceWithDesignerOwnedStructAndWiring(t *testing.T) {
	source := generatedHelloFormSource(defaultHelloFormDesign())
	if _, err := parser.ParseFile(token.NewFileSet(), projectGeneratedFormFile, source, parser.AllErrors); err != nil {
		t.Fatalf("generated form is not valid Go: %v\n%s", err, source)
	}
	text := string(source)
	for _, want := range []string{
		"type MainForm struct",
		"f.messageLabel = forms.NewLabel()",
		"f.helloButton = forms.NewButton()",
		"f.helloButton.Click = f.helloButtonClick",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated form missing %q", want)
		}
	}
}

func TestEmptyDirectoryBecomesHelloWorldProjectWithoutOverwritingGoProjects(t *testing.T) {
	root := t.TempDir()
	created, message := ensureHelloWorldProject(root)
	if !created || message == "" {
		t.Fatalf("project creation = %v, %q", created, message)
	}
	for _, name := range []string{projectMainFile, projectUserFormFile, projectGeneratedFormFile} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("missing generated project file %s: %v", name, err)
		}
	}

	existing := t.TempDir()
	mainPath := filepath.Join(existing, "existing.go")
	if err := os.WriteFile(mainPath, []byte("package existing\n"), 0644); err != nil {
		t.Fatal(err)
	}
	created, _ = ensureHelloWorldProject(existing)
	if created {
		t.Fatal("existing Go project was treated as empty")
	}
	if _, err := os.Stat(filepath.Join(existing, projectGeneratedFormFile)); !os.IsNotExist(err) {
		t.Fatalf("designer file unexpectedly created in existing project: %v", err)
	}
}
