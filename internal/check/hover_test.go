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

func hoverTestOffset(source []byte, marker string) int {
	for i := 0; i+len(marker) <= len(source); i++ {
		if string(source[i:i+len(marker)]) == marker {
			return i + 1
		}
	}
	return -1
}
