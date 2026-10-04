package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestGenericUnsafeDestinationLayouts(t *testing.T) {
	for _, test := range []struct {
		name                    string
		layout                  load.TargetLayout
		alignment, offset, size uint64
	}{
		{"amd64", load.TargetLayout{WordBits: 64, PointerBits: 64}, 8, 8, 16},
		{"386", load.TargetLayout{WordBits: 32, PointerBits: 32}, 4, 4, 12},
		{"wasm32", load.TargetLayout{WordBits: 32, PointerBits: 32, ScalarAlign: 8}, 8, 8, 16},
	} {
		t.Run(test.name, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;func main(){}")}})
			graph.Layout = test.layout
			e := newGenericEnvironment(&graph)
			integer := e.types.basic("uint64")
			record := e.types.intern(genericType{kind: genericStruct, fields: []genericField{{name: "A", typ: e.types.basic("byte")}, {name: "B", typ: integer}}})
			value := e.valueLayout(record, false, 0)
			if !value.known || value.align != int(test.alignment) || value.offsets[1] != test.offset || value.size != test.size {
				t.Fatalf("destination layout=%+v", value)
			}
		})
	}
}
