package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestShiftOperandValidation(t *testing.T) {
	for _, source := range []string{
		`func main(){var n float64=2;_=1<<n}`,
		`func main(){_=1.2<<2}`,
		`func main(){_=1>>2.5}`,
		`type F float32;func f(n F){_=1<<n}`,
		`func f(n complex64){_=n>>1}`,
		`func main(){n:=2.0;_=n<<1}`,
		`func f(n float64){_=1<<n+1}`,
		`func f(n float64){_=1+n<<1}`,
		`func main(){_=true<<1}`,
		`func main(){_=1<<"x"}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrOperand {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestShiftOperandControls(t *testing.T) {
	for _, source := range []string{
		`func main(){_=2.0<<1;_=1<<2.0;_=1<<(2.0)}`,
		`func main(){_=1<<2<<3;_=2*3<<1;_=1<<2*3}`,
		`func main(){_=1<<2+0.5;_=0.5+1<<2}`,
		`func f(n float64){{n:=2;_=1<<n}}`,
		`func f(n float64){_=func(n int)int{return 1<<n}}`,
		`type N uint;func f(n N){_=1<<n}`,
		`func main(){const n=2.0;_=1<<n}`,
		`func main(){_=1<<uint(2)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}
