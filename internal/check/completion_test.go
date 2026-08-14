package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestCompleteGraphResolvesScopeFieldsAndChainedMethods(t *testing.T) {
	mainSource := []byte(`package main

func (f *MainForm) update() {
	localValue := 1
	_ = loc
	ret
	f.mes
	f.messageLabel.SetT
	f.messageLabel.StP
	_ = localValue
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/forms/forms.go", Src: []byte(`package forms
type Label struct{}
func (l *Label) SetText(text string) {}
func (l *Label) SetPosition(x int, y int) {}
func (l *Label) Text() string { return "" }
`)},
		{Path: "/repo/main_form_generated.go", Src: []byte(`package main
import "example.com/app/forms"
type MainForm struct { messageLabel *forms.Label }
`)},
		{Path: "/repo/main_form.go", Src: mainSource},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	if !workspace.Ok {
		t.Fatalf("workspace failed: %#v", workspace)
	}
	assertCompletion := func(marker, want string) {
		t.Helper()
		offset := completionTestOffset(mainSource, marker)
		items := CompleteGraph(workspace.Graph, "/repo/main_form.go", offset)
		for i := 0; i < len(items); i++ {
			if items[i].Name == want {
				return
			}
		}
		t.Fatalf("completion at %q = %#v, want %q", marker, items, want)
	}
	assertCompletion("loc\n", "localValue")
	assertCompletion("ret\n", "return")
	assertCompletion("f.mes\n", "messageLabel")
	assertCompletion("f.messageLabel.SetT\n", "SetText")
	assertCompletion("f.messageLabel.StP\n", "SetPosition")
	items := CompleteGraph(workspace.Graph, "/repo/main_form.go", completionTestOffset(mainSource, "f.messageLabel.SetT\n"))
	for i := 0; i < len(items); i++ {
		if items[i].Name == "SetText" {
			if items[i].Signature != "SetText(text string)" || len(items[i].Parameters) != 1 || items[i].Parameters[0].Name != "text" || items[i].Parameters[0].Type != "string" {
				t.Fatalf("SetText completion signature = %#v", items[i])
			}
			return
		}
	}
	t.Fatal("SetText completion signature was not returned")
}

func TestSignatureHelpGraphReportsMethodParametersAndActiveArgument(t *testing.T) {
	source := []byte(`package main

type Label struct{}
func (l *Label) SetPosition(x int, y int) {}
func update(label *Label) {
	label.SetPosition(10, 20)
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/main.go", Src: source},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	if !workspace.Ok {
		t.Fatalf("workspace failed: %#v", workspace)
	}
	help := SignatureHelpGraph(workspace.Graph, "/repo/main.go", completionTestOffset(source, "10, 20"))
	if !help.Ok || help.Label != "SetPosition(x int, y int)" || help.ActiveParameter != 1 || len(help.Parameters) != 2 {
		t.Fatalf("signature help = %#v", help)
	}
	if help.Parameters[1].Name != "y" || help.Parameters[1].Type != "int" {
		t.Fatalf("active parameter = %#v", help.Parameters[1])
	}
	program := CheckGraph(workspace.Graph)
	cached := SignatureHelpProgram(workspace.Graph, program, "/repo/main.go", completionTestOffset(source, "10, 20"))
	if !cached.Ok || cached.Label != help.Label || cached.ActiveParameter != help.ActiveParameter {
		t.Fatalf("cached signature help = %#v, want %#v", cached, help)
	}
}

func TestCompleteKeywordsWorksWithoutAValidProgram(t *testing.T) {
	items := CompleteKeywords([]byte("func main() { ret"), len("func main() { ret"))
	if len(items) != 1 || items[0].Name != "return" || items[0].Kind != CompletionKeyword {
		t.Fatalf("keyword completion = %#v", items)
	}
}

func TestCompleteProgramInfersMethodResultFromMultiValueAssignment(t *testing.T) {
	mainSource := []byte(`package main
import "example.com/app/sensor"
func main() {
	device := sensor.New()
	value, err := device.Read()
	_ = err
	value.
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/sensor/sensor.go", Src: []byte(`package sensor
type Reading struct { X uint16 }
type Device struct{}
func New() *Device { return &Device{} }
func (d *Device) Read() (Reading, error) { return Reading{}, nil }
`)},
		{Path: "/repo/main.go", Src: mainSource},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	program := CheckGraph(workspace.Graph)
	items := CompleteProgram(workspace.Graph, program, "/repo/main.go", completionTestOffset(mainSource, "value.\n"))
	for i := 0; i < len(items); i++ {
		if items[i].Name == "X" {
			return
		}
	}
	t.Fatalf("completion after inferred method result = %#v", items)
}

func TestCompletionIncludesDeclarationDocumentation(t *testing.T) {
	mainSource := []byte("package main\nimport \"example.com/app/lib\"\nfunc main() { lib.Rea }\n")
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/lib/lib.go", Src: []byte("package lib\n// Read returns the current sample.\nfunc Read() int { return 1 }\n")},
		{Path: "/repo/main.go", Src: mainSource},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	items := CompleteGraph(workspace.Graph, "/repo/main.go", completionTestOffset(mainSource, "lib.Rea"))
	for i := 0; i < len(items); i++ {
		if items[i].Name == "Read" {
			if items[i].Documentation != "Read returns the current sample." {
				t.Fatalf("Read documentation = %q", items[i].Documentation)
			}
			return
		}
	}
	t.Fatal("Read completion was not returned")
}

func completionTestOffset(source []byte, marker string) int {
	for i := 0; i+len(marker) <= len(source); i++ {
		if string(source[i:i+len(marker)]) == marker {
			return i + len(marker) - 1
		}
	}
	return -1
}
