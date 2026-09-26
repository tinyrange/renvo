package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestWideConstGroups(t *testing.T) {
	for _, test := range []struct {
		source string
		valid  bool
	}{
		{"const(A=1<<100;B);func main(){_=append([]byte{},B)}", false},
		{"const(A=iota+255;B);func main(){_=append([]byte{},B)}", false},
		{"const(A,B=iota+254,iota+255;C,D);func main(){_=append([]byte{},C)}", true},
		{"const(A,B=iota+254,iota+255;C,D);func main(){_=append([]byte{},D)}", false},
		{"const(_=iota; A; B=1<<100; C=iota);func main(){_=append([]byte{},C)}", true},
		{"const(A=iota+255;B);const C=iota;func main(){_=append([]byte{},C)}", true},
		{"const iota=256;const(A=iota);func main(){_=append([]byte{},A)}", false},
		{"const(A=1<<100;B);func main(){B:=1;_=append([]byte{},B)}", false},
		{"const(A=1<<100;B);func main(){B:=byte(1);_=append([]byte{},B)}", true},
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + test.source)}})
		result := CheckGraphCore(graph)
		if result.Ok != test.valid || !test.valid && result.Error != CheckErrBuiltinOperand {
			t.Fatalf("%s: ok=%v error=%d", test.source, result.Ok, result.Error)
		}
	}
}
