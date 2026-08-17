package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestNavigateProgramFindsImportedSymbolsAndReferences(t *testing.T) {
	mainSource := []byte(`package main
import "example.com/app/lib"
func main() {
	value := lib.Message()
	_ = value
	_ = lib.Message
	var item lib.Thing
	_ = item
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/lib/lib.go", Src: []byte("package lib\nfunc Message() int { return 1 }\ntype Thing struct{}\n")},
		{Path: "/repo/main.go", Src: mainSource},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	if !workspace.Ok {
		t.Fatalf("workspace failed: %#v", workspace)
	}
	program := CheckGraph(workspace.Graph)
	if !program.Ok {
		t.Fatalf("check failed: %#v", program)
	}
	message := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(mainSource, "Message()"))
	if !message.Ok || message.Definition.Path != "/repo/lib/lib.go" || len(message.References) != 3 {
		t.Fatalf("Message navigation = %#v", message)
	}
	thing := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(mainSource, "Thing"))
	if !thing.Ok || thing.Definition.Path != "/repo/lib/lib.go" || len(thing.References) != 2 {
		t.Fatalf("Thing navigation = %#v", thing)
	}
}

func TestNavigateProgramFindsLocalDeclarationAndUses(t *testing.T) {
	source := []byte(`package main
func main() {
	value := 1
	value = value + 1
	print(value)
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/main.go", Src: source},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	program := CheckGraph(workspace.Graph)
	result := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(source, "value +"))
	if !result.Ok || result.Definition.Path != "/repo/main.go" || len(result.References) != 4 {
		t.Fatalf("local navigation = %#v", result)
	}
}

func TestNavigateProgramFindsFieldsAndMethods(t *testing.T) {
	source := []byte(`package main
type Label struct { Text string }
func (label *Label) SetText(value string) { label.Text = value }
func main() {
	var label Label
	label.SetText("ready")
	print(label.Text)
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/main.go", Src: source},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	program := CheckGraph(workspace.Graph)
	method := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(source, "SetText(\"ready"))
	if !method.Ok || len(method.References) != 2 {
		t.Fatalf("method navigation = %#v", method)
	}
	field := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(source, "Text)"))
	if !field.Ok || len(field.References) != 3 {
		t.Fatalf("field navigation = %#v", field)
	}
}

func TestNavigateProgramFindsImportedConstructorResultMembers(t *testing.T) {
	mainSource := []byte(`package main
import "example.com/app/widgets"
func main() {
	label := widgets.NewLabel()
	label.SetBounds(1)
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/widgets/widgets.go", Src: []byte(`package widgets
type Control struct{}
func (control *Control) SetBounds(value int) {}
type Label struct { Control }
func NewLabel() *Label { return &Label{} }
`)},
		{Path: "/repo/main.go", Src: mainSource},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	program := CheckGraph(workspace.Graph)
	result := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(mainSource, "SetBounds(1"))
	if !result.Ok || result.Definition.Path != "/repo/widgets/widgets.go" || len(result.References) != 2 {
		t.Fatalf("promoted method navigation = %#v, program = %#v", result, program)
	}
}

func TestNavigateProgramFollowsImportedPackageVariables(t *testing.T) {
	mainSource := []byte(`package main
import "example.com/app/board"
func main() { board.Clock.DelayMilliseconds(1) }
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/clock/clock.go", Src: []byte(`package clock
type Clock struct{}
func New() Clock { return Clock{} }
func (c *Clock) DelayMilliseconds(milliseconds uint32) {}
`)},
		{Path: "/repo/board/board.go", Src: []byte(`package board
import "example.com/app/clock"
var Clock = clock.New()
`)},
		{Path: "/repo/main.go", Src: mainSource},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	program := CheckGraph(workspace.Graph)
	result := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(mainSource, "DelayMilliseconds"))
	if !result.Ok || result.Definition.Path != "/repo/clock/clock.go" || len(result.References) != 2 {
		t.Fatalf("package variable method navigation = %#v", result)
	}
}

func TestNavigateProgramWorksWithBestEffortCheck(t *testing.T) {
	source := []byte(`package main
var Value = 1
func use() { println(Value) }
func broken() { unused := 1 }
func main() { use() }
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/main.go", Src: source},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	if checked := CheckGraph(workspace.Graph); checked.Ok {
		t.Fatal("ordinary check unexpectedly accepted unused local")
	}
	program := CheckGraphBestEffort(workspace.Graph)
	result := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(source, "Value)"))
	if !result.Ok || result.Definition.Path != "/repo/main.go" {
		t.Fatalf("best-effort navigation = %#v", result)
	}
}

func TestNavigateProgramFindsMethodInsideFailingBody(t *testing.T) {
	mainSource := []byte(`package main
import "example.com/app/sensor"
func main() {
	device := sensor.New()
	var reading sensor.Reading
	if err := device.ReadInto(&reading); err != nil {}
	unused := 1
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/sensor/sensor.go", Src: []byte(`package sensor
type Reading struct{}
type Device struct{}
func New() *Device { return &Device{} }
func (d *Device) ReadInto(result *Reading) error { return nil }
`)},
		{Path: "/repo/main.go", Src: mainSource},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	if checked := CheckGraph(workspace.Graph); checked.Ok {
		t.Fatal("ordinary check unexpectedly accepted unused local")
	}
	program := CheckGraphBestEffort(workspace.Graph)
	result := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(mainSource, "ReadInto"))
	if !result.Ok || result.Definition.Path != "/repo/sensor/sensor.go" {
		t.Fatalf("method navigation in failing body = %#v", result)
	}
}

func TestNavigateProgramFindsDefinitionsInDependencyModules(t *testing.T) {
	mainSource := []byte(`package main
import "example.com/lib"
func main() { println(lib.Value()) }
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/cache/lib/go.mod", Src: []byte("example.com/lib")},
		{Path: "/cache/lib/lib.go", Src: []byte("package lib\nfunc Value() int { return 42 }\n")},
		{Path: "/repo/main.go", Src: mainSource},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	if !workspace.Ok {
		t.Fatalf("workspace failed: %#v", workspace)
	}
	program := CheckGraph(workspace.Graph)
	result := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(mainSource, "Value()"))
	if !result.Ok || result.Definition.Path != "/cache/lib/lib.go" {
		t.Fatalf("dependency definition = %#v", result)
	}
	packageResult := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(mainSource, "lib.Value"))
	if !packageResult.Ok || packageResult.Definition.Path != "/cache/lib/lib.go" || len(packageResult.References) != 2 {
		t.Fatalf("dependency package definition = %#v", packageResult)
	}
}

func navigationTestOffset(source []byte, marker string) int {
	for i := 0; i+len(marker) <= len(source); i++ {
		if string(source[i:i+len(marker)]) == marker {
			return i + 1
		}
	}
	return -1
}
