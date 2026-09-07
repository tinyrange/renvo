package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestNumericBuiltinValidation(t *testing.T) {
	for _, source := range []string{
		`func main(){_=imag("x")}`,
		`func main(){_=real(("x"))}`,
		`func main(){_=complex("x","y")}`,
		`func main(){_=complex(true,1)}`,
		`func main(){v:=1;_=real(v)}`,
		`func f(v int){_=imag(v)}`,
		`func main(){_=complex(int(1),0)}`,
		`func main(){var a float32;var b float64;_=complex(a,b)}`,
		`type F float32;func main(){var a F;var b float32;_=complex(a,b)}`,
		`func main(){v:=0i;_=complex(v,0)}`,
		`var v="x";func main(){_=imag(v)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrBuiltinOperand {
			t.Fatalf("%s: ok=%v error=%d token=%d", source, result.Ok, result.Error, result.ErrorToken)
		}
	}
}

func TestNumericBuiltinControls(t *testing.T) {
	for _, source := range []string{
		`func main(){_=real(1);_=imag(1.0);_=complex(1,2)}`,
		`func main(){v:=1+2i;_=real(v);_=imag(v)}`,
		`func main(){var v complex64;_=real(v);_=imag(v)}`,
		`type C complex128;func f(v C){_=imag(v)}`,
		`type F float32;func f(a,b F){_=complex(a,b)}`,
		`type F=float32;func f(a F,b float32){_=complex(a,b)}`,
		`func main(){a:=1.0;b:=2e0;_=complex(a,b)}`,
		`func main(){var a float32;_=complex(a,2)}`,
		`const N=2;func main(){_=imag(N)}`,
		`func imag(s string)int{return len(s)};func main(){_=imag("x")}`,
		`func main(){complex:=func(a,b string)int{return len(a)+len(b)};_=complex("x","y")}`,
		`func f(v int){{v:=2i;_=imag(v)};_=v}`,
		`func f(v int){_=func(v complex64){_=imag(v)}}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}
