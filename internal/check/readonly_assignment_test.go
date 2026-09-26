package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestReadOnlyAssignment(t *testing.T) {
	for _, source := range []string{
		`const x=1;func main(){x=2}`,
		`func main(){const x=1;x=2}`,
		`func main(){const(x=1;y=2);x,y=3,4}`,
		`const x=1;func main(){x+=2}`,
		`func f()int{return 1};func main(){f()=2}`,
		`func f()int{return 1};func main(){(f())=2}`,
		`const x=1;func main(){x++}`,
		`func main(){const x=1;x--}`,
		`func f()int{return 1};func main(){f()++}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrAssignTarget {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestReadOnlyAssignmentControls(t *testing.T) {
	for _, source := range []string{
		`const x=1;func f(x int){x=2;_=x}`,
		`const x=1;func main(){x:=2;x=3;_=x}`,
		`func main(){const x=1;{x:=2;x=3;_=x}}`,
		`func main(){x:=1;{const x=2;_=x};x=3;_=x}`,
		`const x=1;func main(){if x:=2;x>0{x=3;_=x}}`,
		`func main(){const x=1;_=func(x int){x=2}}`,
		`func f()*int{return new(int)};func main(){*f()=2}`,
		`func f()[]int{return []int{1}};func main(){f()[0]=2}`,
		`func main(){x:=1;(x)=2;_=x}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}
