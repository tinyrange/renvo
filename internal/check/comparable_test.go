package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestMapKeyComparability(t *testing.T) {
	for _, item := range []struct {
		source string
		valid  bool
	}{
		{`var m map[[]int]int`, false},
		{`type K []int; var m map[K]int`, false},
		{`var m map[[2][]int]int`, false},
		{`var m map[struct{ A []int }]int`, false},
		{`var m map[func()]int`, false},
		{`var m map[map[int]int]int`, false},
		{`type K struct{ A [2]int }; var m map[K]int`, true},
		{`var m map[*[]int]int`, true},
		{`var m map[interface{}]int`, true},
		{`var m map[chan int]int`, true},
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + item.source + "\nfunc main(){}")}})
		result := CheckGraphCore(graph)
		if result.Ok != item.valid || !item.valid && result.Error != CheckErrMapKey {
			t.Fatalf("%s: ok=%v error=%d", item.source, result.Ok, result.Error)
		}
	}
}
