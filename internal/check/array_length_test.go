package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestInvalidPackageArrayLengths(t *testing.T) {
	for _, source := range []string{
		"var x [-1]int",
		"var x [1.5]int",
		"var x [1<<100]int",
		"var x [9223372036854775808]int",
		"var x [(1<<100)-(1<<100)-1]int",
		"var x [((1<<64)+1)*((1<<64)-1)]int",
		"const n=-1; var x [n]int",
		"const n=1<<100; type T [n]int",
		"type T struct { X [1<<100]int }",
		"var x map[string][1<<100]int",
		"var x *[1<<100]int",
	} {
		t.Run(source, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source + "\nfunc main(){}")}})
			result := CheckGraphCore(graph)
			if result.Ok || result.Error != CheckErrArrayLength {
				t.Fatalf("ok=%v error=%d", result.Ok, result.Error)
			}
		})
	}
}

func TestArrayLengthLargeIntermediates(t *testing.T) {
	for _, source := range []string{
		"var x [0]int",
		"var x [1.0]int",
		"var x [(1<<100)>>100]int",
		"var x [(1<<100)-(1<<100)+1]int",
		"const n=1<<100; var x [n-n+2]int",
		"var x [0x10000000000000000-0xFFFFFFFFFFFFFFFF]int",
		"var x [(1<<63)-1]byte",
		"var x map[[1]int][2]int",
	} {
		t.Run(source, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source + "\nfunc main(){}")}})
			if result := CheckGraphCore(graph); !result.Ok {
				t.Fatalf("error=%d token=%d", result.Error, result.ErrorToken)
			}
		})
	}
}
