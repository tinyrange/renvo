package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestArrayLiteralBounds(t *testing.T) {
	for _, source := range []string{
		`func main(){_=[1]int{1,2}}`,
		`func main(){_=[0]int{1}}`,
		`func main(){_=[3]int{3:1}}`,
		`func main(){_=[3]int{2:1,2}}`,
		`func main(){_=[3]int{-1:1}}`,
		`func main(){_=[...]int{-1:1}}`,
		`type A [1]int;type B=A;var _=B{1,2}`,
		`const N=2;func main(){_=[N]int{N:1}}`,
		`func main(){_=[1]int{1<<80:1}}`,
		`func main(){_=[(1<<80)-(1<<80)+1]int{1,2}}`,
		`func main(){const N=1;_=[N]int{1,2}}`,
		`func main(){const N=2;_=[N]int{N:1}}`,
		`func main(){const N=1<<100;_=[1]int{N:1}}`,
		`func main(){const N=-1;_=[...]int{N:1}}`,
		`func main(){const(A=iota;B);_=[1]int{B:1}}`,
		`func main(){const N=2;_=[N]int{N-1:1,2}}`,
		`const N=0;func main(){{const N=2;_=N};_=[N]int{1}}`,
		`const N=0;type A [N]int;func main(){const N=2;_=A{1}}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrArrayIndex {
			t.Fatalf("%s: ok=%v error=%d token=%d", source, result.Ok, result.Error, result.ErrorToken)
		}
	}
}

func TestArrayLiteralBoundsControls(t *testing.T) {
	for _, source := range []string{
		`func main(){_=[3]int{1,2,3}}`,
		`func main(){_=[0]int{}}`,
		`func main(){_=[3]int{2:1,0:2,3}}`,
		`func main(){_=[...]int{5:1,2}}`,
		`func main(){_=[]int{1,2,3}}`,
		`const N=0;func main(){const N=2;_=[N]int{1,2}}`,
		`type A [1]int;func main(){type A [2]int;_=A{1,2}}`,
		`type A [2]int;type B=A;var _=B{1,2}`,
		`func main(){_=[1]int{(1<<80)-(1<<80):1}}`,
		`func main(){_=[2]int{0.0:1,2}}`,
		`func main(){const N=2;_=[N]int{N-1:1}}`,
		`const N=1;func main(){const N=N+1;_=[N]int{1,2}}`,
		`func main(){const N=2;const M=N;{const N=0;_=[M]int{1,2}}}`,
		`func main(){const N=0;_=func(){const N=2;_=[N]int{1,2}}}`,
		`func main(){const N=2;_=func(){const N=0;_=[2]int{N:1}}}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}
