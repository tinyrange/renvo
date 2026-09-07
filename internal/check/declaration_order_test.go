package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestLocalDeclarationOrder(t *testing.T) {
	for _, source := range []string{
		`func main(){_=x;x:=1;_=x}`,
		`func main(){_=x;var x int;_=x}`,
		`func main(){_=x;const x=1;_=x}`,
		`func main(){var x T;type T int;_=x}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrUndefined {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestLocalDeclarationOrderControls(t *testing.T) {
	for _, source := range []string{
		`var x=2;func main(){_=x;x:=1;_=x}`,
		`func main(){_=len("abc");len:=1;_=len}`,
		`func main(){goto done;done: return}`,
		`func main(){type Node struct{next *Node};var n Node;_=n}`,
		`func main(){x:=1;{_=x;x:=2;_=x}}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}
