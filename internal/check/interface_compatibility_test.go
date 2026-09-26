package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestDefiniteInterfaceCompatibility(t *testing.T) {
	for _, source := range []string{
		`type I interface{M(int)};type T int;func(T)M(string){};func main(){var _ I=T(0)}`,
		`type I interface{M(int)};type T int;func(T)M(string){};var _ I=T(0);func main(){}`,
		`type I interface{M(int)};type T int;func(T)M(string){};func main(){var x I;x=T(0);_=x}`,
		`type I interface{M()};type T int;func main(){var x I;_=x.(T)}`,
		`type I interface{M(int)};type T int;func(T)M(string){};func main(){var x I;_=x.(T)}`,
		`type I interface{M()};type T int;func(*T)M(){};func main(){var _ I=T(0)}`,
		`type I interface{M()};type T struct{X int};func main(){var _ I=T{}}`,
		`type I interface{M()int};type T int;func(T)M()string{return ""};func main(){var _ I=T(0)}`,
		`type I interface{M(...int)};type T int;func(T)M([]int){};func main(){var _ I=T(0)}`,
		`type N int;type I interface{M(N)};type T int;func(T)M(int){};func main(){var _ I=T(0)}`,
		`type I interface{M()};type T int;type A=T;func main(){var x I;_=x.(A)}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrType {
			t.Fatalf("%s: ok=%v error=%d token=%d", source, result.Ok, result.Error, result.ErrorToken)
		}
	}
}

func TestDefiniteInterfaceCompatibilityControls(t *testing.T) {
	for _, source := range []string{
		`type I interface{M(int)};type T int;func(T)M(x int){};func main(){var _ I=T(0)}`,
		`type I interface{M(byte)};type T int;func(T)M(uint8){};func main(){var _ I=T(0)}`,
		`type N=int;type I interface{M(N)};type T int;func(T)M(int){};func main(){var _ I=T(0)}`,
		`type I interface{M()};type T int;func(*T)M(){};func main(){var t T;var _ I=&t}`,
		`type I interface{M()};type T int;func(T)M(){};type S struct{T};func main(){var _ I=S{}}`,
		`type I interface{M()};type T int;func(*T)M(){};type S struct{*T};func main(){var _ I=S{}}`,
		`type I interface{M()};type T int;func main(){type I interface{};var _ I=T(0)}`,
		`type I interface{M()};type T int;func(T)M(){};func main(){var x I=T(1);_=x.(T)}`,
		`type I interface{M()};type T int;func main(){var x I;_=x;var obj struct{x any};_=obj.x.(T)}`,
		`type I interface{M()};type T int;func main(){var x I;_=x;_=func(x any){_=x.(T)}}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}
