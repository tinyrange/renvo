package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestPointerOrdering(t *testing.T) {
	for _, source := range []string{
		`func main(){a:=new(int);b:=new(int);_=a<b}`,
		`func main(){a:=1;b:=2;_=&a>=&b}`,
		`func f(a,b *int){_=a<=b}`,
		`func main(){var a,b *int;_=a>b}`,
		`type P *int; type Q = P; func f(a,b Q){_=a<b}`,
		`func pointer()*int{return nil};func main(){_=pointer()<pointer()}`,
		`func main(){a:=new(int);b:=a;_=b<a}`,
		`var a = new(int);var b *int;func main(){_=a<b}`,
		`func main(){a:=new(*int);b:=new(*int);_=*a<*b}`,
		`func main(){_=(*int)(nil)<(*int)(nil)}`,
		`func main(){a:=new(int);_=a<nil}`,
		`func f(a *int){{a:=1;_=a<2};_=a<a}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrOperand {
			t.Fatalf("%s: ok=%v error=%d token=%d", source, result.Ok, result.Error, result.ErrorToken)
		}
	}
}

func TestPointerOrderingControls(t *testing.T) {
	for _, source := range []string{
		`func main(){a:=new(int);b:=new(int);_=a==b;_=a!=nil;_=*a<*b}`,
		`func f(a,b **int){_=**a<**b}`,
		`func f(a *int){{a:=1;_=a<2};_=a==nil}`,
		`var a *int;func main(){a:=1;_=a<2}`,
		`func new(x int)int{return x};func main(){a:=new(1);_=a<2}`,
		`func main(){new:=func(x int)int{return x};a:=new(1);_=a<2}`,
		`type P *int;func main(){type P int;var a,b P;_=a<b}`,
		`func main(){a:=new(int);if a:=1;a<2{_=a<3};_=a==nil}`,
		`func main(){a:=1;b:=2;_=(a&b)<3;_=(a*b)<3}`,
		`func main(){a:=1;p:=&a;_=*p+1<3}`,
		`func f(a *int){_=func(a int)bool{return a<2}}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}
