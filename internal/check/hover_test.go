package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestHoverProgramReportsInferredTypesAndDocumentation(t *testing.T) {
	mainSource := []byte(`package main
import "example.com/app/sensor"
func main() {
	device := sensor.New()
	value, err := device.Read()
	count := int16(3)
	_ = err
	_ = count
	print(value.X)
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/sensor/sensor.go", Src: []byte(`package sensor
type Reading struct {
	// X is the acceleration on the X axis.
	X uint16
}
type Device struct{}
func New() *Device { return &Device{} }
// Read returns the latest sample.
func (d *Device) Read() (Reading, error) { return Reading{}, nil }
`)},
		{Path: "/repo/main.go", Src: mainSource},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	program := CheckGraph(workspace.Graph)
	if !program.Ok {
		t.Fatalf("check failed: %#v", program)
	}
	value := HoverProgram(workspace.Graph, program, "/repo/main.go", hoverTestOffset(mainSource, "value.X"))
	if !value.Ok || value.Signature != "var value sensor.Reading" {
		t.Fatalf("value hover = %#v", value)
	}
	count := HoverProgram(workspace.Graph, program, "/repo/main.go", hoverTestOffset(mainSource, "count :="))
	if !count.Ok || count.Signature != "var count int16" {
		t.Fatalf("count hover = %#v", count)
	}
	read := HoverProgram(workspace.Graph, program, "/repo/main.go", hoverTestOffset(mainSource, "Read()"))
	if !read.Ok || read.Signature != "func Read() (Reading, error)" || read.Documentation != "Read returns the latest sample." {
		t.Fatalf("Read hover = %#v", read)
	}
	field := HoverProgram(workspace.Graph, program, "/repo/main.go", hoverTestOffset(mainSource, "X)"))
	if !field.Ok || field.Signature != "field X uint16" || field.Documentation != "X is the acceleration on the X axis." {
		t.Fatalf("X hover = %#v", field)
	}
}

func TestHoverProgramReportsChannelAndReceiveTypes(t *testing.T) {
	source := []byte(`package main
func main() {
	values := make(chan int, 1)
	value := <-values
	_ = value
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/main.go", Src: source},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	program := CheckGraph(workspace.Graph)
	if !program.Ok {
		t.Fatalf("check failed: %#v", program)
	}
	values := HoverProgram(workspace.Graph, program, "/repo/main.go", hoverTestOffset(source, "values :="))
	if !values.Ok || values.Signature != "var values chan int" {
		t.Fatalf("values hover = %#v", values)
	}
	value := HoverProgram(workspace.Graph, program, "/repo/main.go", hoverTestOffset(source, "value :="))
	if !value.Ok || value.Signature != "var value int" {
		t.Fatalf("value hover = %#v", value)
	}
}

func TestHoverProgramReportsConstantsPackagesAndBuiltins(t *testing.T) {
	mainSource := []byte(`package main
import "example.com/app/board"
const register = 0x1f
func main() {
	const mask = 16
	println(board.Value, register, mask)
}
`)
	files := []load.SourceFile{
		{Path: "/repo/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/board/board.go", Src: []byte(`// Package board exposes the test hardware.
package board
const Value = 1
`)},
		{Path: "/repo/main.go", Src: mainSource},
	}
	workspace := load.LoadWorkspace("/repo", "/std", ".", files)
	program := CheckGraph(workspace.Graph)
	constant := HoverProgram(workspace.Graph, program, "/repo/main.go", hoverTestOffset(mainSource, "register,"))
	if !constant.Ok || constant.Signature != "const register = 31 // 0x1f" {
		t.Fatalf("constant hover = %#v", constant)
	}
	local := HoverProgram(workspace.Graph, program, "/repo/main.go", hoverTestOffset(mainSource, "mask)"))
	if !local.Ok || local.Signature != "const mask = 16 // 0x10" {
		t.Fatalf("local constant hover = %#v", local)
	}
	pkg := HoverProgram(workspace.Graph, program, "/repo/main.go", hoverTestOffset(mainSource, "board.Value"))
	if !pkg.Ok || pkg.Signature != `package board // "example.com/app/board"` || pkg.Documentation != "Package board exposes the test hardware." {
		t.Fatalf("package hover = %#v", pkg)
	}
	builtin := HoverProgram(workspace.Graph, program, "/repo/main.go", hoverTestOffset(mainSource, "println("))
	if !builtin.Ok || builtin.Signature == "" || builtin.Documentation == "" {
		t.Fatalf("builtin hover = %#v", builtin)
	}
}

func hoverTestOffset(source []byte, marker string) int {
	for i := 0; i+len(marker) <= len(source); i++ {
		if string(source[i:i+len(marker)]) == marker {
			return i + 1
		}
	}
	return -1
}
