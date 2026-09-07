package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestLocalArrayLengthDeclarations(t *testing.T) {
	for _, source := range []string{
		`func main(){var a [-1]int;_=a}`,
		`func main(){var a [1.5]int;_=a}`,
		`func main(){var a [true]int;_=a}`,
		`func main(){var a [nil]int;_=a}`,
		`func main(){n:=2;var a [(n)]int;_=a}`,
		`func f(n int){var a [n]int;_=a};func main(){}`,
		`const n=2;func main(){n:=2;var a [n]int;_=a}`,
		`func main(){const N="size";var a [N]int;_=a}`,
		`func main(){const(N=false;M);type A [M]int}`,
		`type Flag bool;func main(){const N Flag=false;var a [N]int;_=a}`,
		`func main(){const N=1<<100;var a [N]int;_=a}`,
		`func main(){const N=-1;type A [N]int}`,
		`func main(){const N=-1;type A = [N]int}`,
		`func main(){const N=-1;type(A [N]int)}`,
		`func main(){const N=-1;var(a [N]int);_=a}`,
		`func main(){const N=-1;var a,b [N]int;_,_=a,b}`,
		`func main(){const N=-1;var a map[string]*[N]int;_=a}`,
		`func main(){const N=-1;type A struct{Field [N]int}}`,
		`func main(){const N=1<<100;var a [(N-N)-1]int;_=a}`,
		`func main(){const(_=iota;N);var a [-N]int;_=a}`,
		`const N=-1;func main(){{const N=2;_=N};var a [N]int;_=a}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		result := CheckGraphCore(graph)
		if result.Ok || result.Error != CheckErrArrayLength {
			t.Errorf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestLocalArrayLengthDeclarationControls(t *testing.T) {
	for _, source := range []string{
		`func main(){const N=2;var a [N]int;_=a}`,
		`func main(){const N=2;type A [N]int}`,
		`func main(){const N=2;type A = [N]int}`,
		`func main(){const N=1<<100;var a [N-N+2]int;_=a}`,
		`const N=-1;func main(){const N=N+3;var a [N]int;_=a}`,
		`func main(){const N=2;const M=N;{const N=-1;var a [M]int;_=a}}`,
		`func main(){const N=-1;_=func(){const N=2;var a [N]int;_=a}}`,
		`func main(){var a map[[2]int][3]int;_=a}`,
		`func main(){var a [1.0]int;_=a}`,
		`const N="size";func main(){const N=2;var a [N]int;_=a}`,
		`const N=2;func main(){{const N="size";_=N};var a [N]int;_=a}`,
		`func main(){const N=2;const M=N;{const N="size";var a [M]int;_=a}}`,
		`func main(){const true=2;var a [true]int;_=a}`,
		`func main(){const nil=2;var a [nil]int;_=a}`,
		`var n=2;func main(){const n=2;var a [n]int;_=a}`,
		`func main(){n:=2;{const n=2;var a [n]int;_=a};_=n}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Errorf("%s: error=%d", source, result.Error)
		}
	}
}
