package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestInvalidKnownConversions(t *testing.T) {
	for _, source := range []string{
		`func main(){_=[]int("x")}`,
		`func main(){_=int(struct{}{})}`,
		`func main(){_=string(1.2)}`,
		`type N int;type Ns []N;func main(){_=Ns("x")}`,
		`type Text string;func f(v float64){_=Text(v)}`,
		`func main(){_=int("x")}`,
		`func main(){_=bool(1)}`,
		`func main(){_=float64(true)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrOperand {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestKnownConversionControls(t *testing.T) {
	for _, source := range []string{
		`func main(){_=[]byte("x");_=[]rune("x");_=string(65)}`,
		`type B byte;type Bs []B;func main(){_=Bs("x")}`,
		`func main(){var f float64=1.2;_=int(f);_=float64(1);_=complex128(1)}`,
		`func main(){_=bool(true);_=[]int(nil)}`,
		`func main(){int:=func(s string)int{return len(s)};_=int("x")}`,
		`func string(v float64)int{return 1};func main(){_=string(1.2)}`,
		`func main(){type string int;_=string(1)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}
