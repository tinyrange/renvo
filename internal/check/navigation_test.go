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

func TestNavigateProgramFindsSelectReceiveBinding(t *testing.T) {
	source := []byte(`package main
func main() {
	values := make(chan int, 1)
	select {
	case received := <-values:
		print(received)
	}
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/main.go", Src: source},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	program := CheckGraph(workspace.Graph)
	result := NavigateProgram(workspace.Graph, program, "/repo/main.go", navigationTestOffset(source, "received)"))
	if !result.Ok || result.Definition.Path != "/repo/main.go" || len(result.References) != 2 {
		t.Fatalf("select receive navigation = %#v", result)
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

func navigationTestOffset(source []byte, marker string) int {
	for i := 0; i+len(marker) <= len(source); i++ {
		if string(source[i:i+len(marker)]) == marker {
			return i + 1
		}
	}
	return -1
}
