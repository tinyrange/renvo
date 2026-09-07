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
		"var x [(1<<100)/2]int",
		"var x [-257%129]int",
		"var x [(1<<100)&^1]int",
		"var x [-1&^1]int",
		"var x [-'a']int",
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
		"var x [(1<<100)/(1<<100)]int",
		"var x [(1<<100)%257]int",
		"var x [((1<<100)|3)&3]int",
		"var x [((1<<100)|3)^(1<<100)]int",
		"var x ['界'-'界'+'a'-96]int",
	} {
		t.Run(source, func(t *testing.T) {
			graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source + "\nfunc main(){}")}})
			if result := CheckGraphCore(graph); !result.Ok {
				t.Fatalf("error=%d token=%d", result.Error, result.ErrorToken)
			}
		})
	}
}
