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
	source := generatedFormSource(defaultFormDesign())
	if _, err := parser.ParseFile(token.NewFileSet(), projectGeneratedFormFile, source, parser.AllErrors); err != nil {
		t.Fatalf("generated form is not valid Go: %v\n%s", err, source)
	}
	text := string(source)
	for _, want := range []string{
		"type MainForm struct",
		"f.uiFont = gofont.New(15)",
		"f.messageLabel = forms.NewLabel()",
		`f.messageLabel.SetAccessibilityID("MainForm.messageLabel")`,
		"f.helloButton = forms.NewButton()",
		`f.helloButton.SetAccessibilityID("MainForm.helloButton")`,
		"f.helloButton.Click = f.helloButtonClick",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated form missing %q", want)
		}
	}
}

func TestGeneratedFormIsTheRoundTrippableDesignerDocument(t *testing.T) {
	design := formDesign{
		width:        640,
		height:       360,
		paintHandler: "mainFormPaint",
		controls: []designerControl{
			{kind: designerLabel, name: "statusLabel", text: "Ready\nnow", x: 17, y: 21, width: 180, height: 30},
			{kind: designerButton, name: "launchButton", text: "Launch \"app\"", x: 220, y: 260, width: 140, height: 42, clickHandler: "launchButtonClick"},
			{kind: designerTextBox, name: "nameTextBox", text: "Ada", x: 20, y: 70, width: 180, height: 32},
			{kind: designerTextArea, name: "notesTextArea", text: "one\ntwo", x: 20, y: 110, width: 220, height: 90},
			{kind: designerCheckBox, name: "enabledCheckBox", text: "Enabled", x: 270, y: 70, width: 140, height: 28, checked: true},
			{kind: designerRadioButton, name: "modeRadioButton", text: "Mode", x: 270, y: 105, width: 140, height: 28},
			{kind: designerPictureBox, name: "logoPictureBox", x: 450, y: 30, width: 120, height: 90, paintHandler: "logoPictureBoxPaint"},
			{kind: designerPanel, name: "settingsPanel", x: 260, y: 150, width: 250, height: 100},
		},
	}
	source := generatedFormSource(design)
	parsed, message := parseFormDesign(source)
	if message != "" {
		t.Fatal(message)
	}
	if parsed.width != design.width || parsed.height != design.height || len(parsed.controls) != len(design.controls) {
		t.Fatalf("parsed design = %#v", parsed)
	}
	for i := 0; i < len(design.controls); i++ {
		if parsed.controls[i] != design.controls[i] {
			t.Fatalf("control %d = %#v, want %#v", i, parsed.controls[i], design.controls[i])
		}
	}
	if regenerated := generatedFormSource(parsed); string(regenerated) != string(source) {
		t.Fatal("designer generation was not deterministic after parsing")
	}
}

func TestGeneratedFormRoundTripsEveryDesignerControl(t *testing.T) {
	design := formDesign{width: 900, height: 700}
	for i := 0; i < len(designerControlKinds); i++ {
		value := 0
		if designerControlHasValue(designerControlKinds[i]) {
			value = 40
		}
		label := ""
		if designerControlHasText(designerControlKinds[i]) {
			label = designerControlNames[i]
		}
		dock := ""
		if designerControlKinds[i] == designerMenuBar {
			dock = designerDockTop
		} else if designerControlKinds[i] == designerStatusBar {
			dock = designerDockBottom
		}
		design.controls = append(design.controls, designerControl{
			kind: designerControlKinds[i], name: designerControlKinds[i] + "1", text: label,
			dock: dock, x: 10 + i*3, y: 10 + i*4, width: 180, height: 36, value: value,
		})
	}
	source := generatedFormSource(design)
	if _, err := parser.ParseFile(token.NewFileSet(), projectGeneratedFormFile, source, parser.AllErrors); err != nil {
		t.Fatalf("full-palette source is invalid: %v\n%s", err, source)
	}
	parsed, message := parseFormDesign(source)
	if message != "" {
		t.Fatal(message)
	}
	if !strings.Contains(string(source), ".SetDock(forms.DockTop)") || !strings.Contains(string(source), ".SetDock(forms.DockBottom)") {
		t.Fatal("generated menu and status controls did not retain docking")
	}
	if len(parsed.controls) != len(designerControlKinds) {
		t.Fatalf("round trip controls = %d, want %d", len(parsed.controls), len(designerControlKinds))
	}
	for i := 0; i < len(design.controls); i++ {
		if parsed.controls[i] != design.controls[i] {
			t.Fatalf("control %d = %#v, want %#v", i, parsed.controls[i], design.controls[i])
		}
	}
}

func TestEmptyDirectoryBecomesHelloWorldProjectWithoutOverwritingGoProjects(t *testing.T) {
	root := t.TempDir()
	created, message := ensureHelloWorldProject(root)
	if !created || message == "" {
		t.Fatalf("project creation = %v, %q", created, message)
	}
	for _, name := range []string{projectModuleFile, projectMainFile, projectUserFormFile, projectGeneratedFormFile} {
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
