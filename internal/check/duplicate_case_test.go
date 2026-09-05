package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestDuplicateSwitchCases(t *testing.T) {
	for _, source := range []string{
		`func main(){switch 1 {case 1: case 1:}}`,
		`func main(){switch 1 {case 1, 0x1:}}`,
		`func main(){switch -2 {case -2: case (-0b10):}}`,
		`func main(){switch "a" {case "a": case "\x61":}}`,
		`func main(){switch 1 {default: default:}}`,
		`func f(x interface{}){switch x.(type) {case int: case int:}}`,
		`func f(x interface{}){switch x.(type) {case []int: case []int:}}`,
		`var m = map[int]int{1: 2, 0x1: 3}`,
		`func main(){_ = map[string]int{"a": 1, "\x61": 2}}`,
		`func main(){_ = map[int]map[int]int{1: {1: 2}, 1: {2: 3}}}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		result := CheckGraphCore(graph)
		if result.Ok || result.Error != CheckErrDuplicate {
			t.Errorf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestIndependentSwitchCases(t *testing.T) {
	for _, source := range []string{
		`func main(){switch 1 {case 1: switch 1 {case 1: default:}; default:}}`,
		`func main(){switch 1 {case 1:}; switch 1 {case 1:}}`,
		`func f(a,b int){switch 1 {case a: case b:}}`,
		`func f(x interface{}){switch x {case int(1): case int8(1):}}`,
		`func f(x interface{}){switch x.(type) {case int: case []int:}}`,
		`func f(x uint64){switch x {case 0: case 18446744073709551616-1:}}`,
		`func f(x bool){switch x {case true: case true:}}`,
		`func f(a,b int){_ = map[int]int{a: 1, b: 2}}`,
		`func main(){_ = map[interface{}]int{int(1): 1, int8(1): 2}}`,
		`func main(){_ = map[int]map[int]int{1: {1: 2}, 2: {1: 3}}}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		result := CheckGraphCore(graph)
		if !result.Ok {
			t.Errorf("%s: error=%d", source, result.Error)
		}
	}
}
