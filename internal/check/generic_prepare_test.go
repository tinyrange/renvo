package check

import (
	"renvo.dev/internal/load"
	"strings"
	"testing"
)

func TestGenericPreparationLanguageCases(t *testing.T) {
	for _, source := range []string{
		`func Box[T any](v T)any{return v};func Matches[T any](v any)bool{_,ok:=v.(T);return ok};func main(){var f func(interface{Read()int})interface{Read()int};if !Matches[func(interface{Read()int})interface{Read()int}](Box(f)){panic(1)}}`,
		`func Truth[T ~bool]()T{const v=!(1<<100 < 1<<99)&&"abc"<"b"&&1i==complex(0,1);return T(v)};func main(){_=Truth[bool]()}`,
		`func Length[T ~int]()T{var a [3]int;const n=len((a));return T(n)};func main(){_=Length[int]()}`,
		`func Length[T ~int]()T{const n=len([3]int{int(1)});return T(n)};func main(){_=Length[int]()}`,
		`func Length[T ~int]()T{const n=cap(([3]int)([3]int{}));return T(n)};func main(){_=Length[int]()}`,
		`func Make()[3]int{return [3]int{}};func Length[T ~int]()T{a:=Make();const n=len(a);return T(n)};func main(){_=Length[int]()}`,
		`func Pair()([3]int,int){return [3]int{},0};func Length[T ~int]()T{a,_:=Pair();const n=len(a);return T(n)};func main(){_=Length[int]()}`,
		`func Length[T ~int]()T{const n=len([3]int{len("abc")});return T(n)};func main(){_=Length[int]()}`,
		`func Recur[T ~int](v T,n int) T {if n==0{return v};return Recur(v,n-1)};func main(){print(Recur(42,2))}`,
		`func Recur[S ~[]E,E any](s S,n int) S {if n==0{return s};return Recur(s,n-1)};func main(){print(Recur([]int{42},2)[0])}`,
		`func Recur[T any](v T,n int) T {if n==0{return v};var f func(T,int)T=Recur;return f(v,n-1)};func main(){print(Recur(42,2))}`,
		`func Shift[T ~int](n uint) T {return 1.0<<n};func main(){print(Shift[int](3))}`,
		`func Shift[T ~int](n uint) T {return (1.0<<n)<<n};func main(){print(Shift[int](2))}`,
		`func Shift[T ~int](n uint) T {return 1<<(1.0<<n)};func main(){print(Shift[int](2))}`,
		`func Shift[T any](n uint) any {return 1<<n};func main(){print(Shift[int](3).(int))}`,
		`func Shift[T ~int8](n uint) T {return (1<<n)+127};func main(){print(Shift[int8](0))}`,
		`func Reuse[T any](v T) T {v,x:=v,2;_=x;{v,x:=1,3;_,_=v,x};return v};func main(){print(Reuse(42))}`,
		`func Bounds[S ~[]E,E any](s S,n int) S{return s[0:n:2]};func main(){print(len(Bounds([]int{1,2},1)))}`,
		`func Bounds[S ~[]E,E any](n int) S{return make(S,n,2)};func main(){print(len(Bounds[[]int](1)))}`,
		`func Bounds[T any](v *[2]T) []T{return v[2:2:2]};func main(){print(len(Bounds(&[2]int{})))}`,
		`func Text[T ~string]() T{return T("abc"[1:])};func main(){print(Text[string]())}`,
		`func Bounds[T ~[2]int](v T) []int{return v[:2:2]};func main(){print(len(Bounds([2]int{})))}`,
		`func Array[T any](v T)[3]T{return [...]T{2:v}};func Identity[T any](v T)T{return v};func main(){print(Array(42)[2]);print(Identity([... ]int{2:42})[2])}`,
		`func Last[T ~[2]int|~[3]int](v T)int{return v[1]};func main(){print(Last([2]int{1,42}))}`,
		`func Shift[T ~int]() T{return (1<<100)>>99};func main(){print(Shift[int]())}`,
		`func Equal[T comparable](v T,w any)bool{return v==w};func main(){print(Equal(42,any(42)))}`,
		`func Shift[T ~int]() T{return 1.0<<3};func main(){print(Shift[int]())}`,
		`func Pointer[T any](v T)*struct{Value T}{return &(struct{Value T}{v})};func main(){print(Pointer(42).Value)}`,
		`func Sparse[T any](v T) []T{return []T{2:v,0:v,v}};func main(){print(len(Sparse(42)))}`,
		`func Letters[T ~int]() T{return len(string(65))+len(string(0x1f600))+len(string(-1))};func main(){print(Letters[int]())}`,
		`func Count[T ~map[int]int](m T){m[0]++;m[1]=42};func main(){m:=map[int]int{};Count(m);print(m[0]+m[1])}`,
		`func Complement[T ~uint8]() T{return T(^uint8(1))};func main(){print(Complement[uint8]())}`,
		`func Equal[T ~bool](v int) T{return v==0};func main(){print(Equal[bool](0))}`,
		`func Parts[T ~int]() T{return real(complex(40,3))+imag(2i)};func Typed[T ~int8]() T{return T(min(int(127),128))};func main(){print(Parts[int]());print(Typed[int8]())}`,
		`func Rounded[T ~int8]() T{return T(float32(127.000001))};func main(){print(Rounded[int8]())}`,
		`func Retag[T ~struct{Value int "old"}](v T) struct{Value int "new"}{return struct{Value int "new"}(v)};func main(){print(Retag(struct{Value int "old"}{42}).Value)}`,
		`func Array[S ~[]E,E any](s S)[2]E{return [2]E(s)};func Pointer[S ~[]E,E any](s S)*[2]E{return (*[2]E)(s)};func main(){print(Array([]int{1,2})[0]);print(Pointer([]int{3,4})[1])}`,
		`const(a=iota;b;c);const(d,e=40+iota,41+iota;f,g);func Number[T ~int]() T{return c+g};func main(){print(Number[int]())}`,
		`const(a int=iota;b;c);func Number[T ~int]() T{return T(c)};func main(){print(Number[int]())}`,
		`func Identity[T any](v T) T{return v};func main(){type Box struct{Value int};f:=func()int{b:=Box{42};return Identity(b).Value};print(f())}`,
		`func Ordinal[T ~int]() T {const(a=iota;b;c);return c};func main(){print(Ordinal[int]())}`,
		`func Classify[T any](v T) int {switch x:=any(v).(type){case int:return x;case string:return len(x);default:return 0}};func main(){print(Classify(42));print(Classify("yes"))}`,
		`func Receive[C ~chan E|~<-chan E,E any](ch C) E{return <-ch};func main(){ch:=make(chan int,1);ch<-42;print(Receive(ch))}`,
		`func Receive[T any](ch <-chan T)(T,bool){select{case value,ok:=<-ch:return value,ok;default:var zero T;return zero,false}};func main(){}`,
		`const Text string="ab";func Length[T ~int]() T {const local="c";return len(Text+local)};func Size[T ~int]() T {var a [3]byte;return cap(a)};func Compare[T ~bool]() T{return 1<2};func main(){print(Length[int]());print(Size[int]());print(Compare[bool]())}`,
		`func Slice[T ~[]byte|~string](v T) T {return v[1:]};func Array[T ~[2]int](v T) []int {return v[:]};func main(){print(Slice("abc"));print(Array([2]int{1,2})[0])}`,
		`func Zero[T any]() func()T { return func() T { var zero T; return zero } }
func Shadow[T any](v T) func()T { return func() T { T:=v;return T } }
func main(){ print(Zero[int]()());print(Shadow(42)()) }`,
		`type Box[T any] struct{Value T};type Alias[T any]=Box[T];type Fixed=Box[int]
type Holder struct{Fixed};type Other[T any] struct{Alias[T]}
func Identity[T any](v T) T { return v }
func main(){ h:=Holder{Fixed:Box[int]{42}};o:=Other[int]{Alias:Box[int]{42}};print(Identity(h).Fixed.Value);print(o.Alias.Value);a:=struct{Alias[int]}{Box[int]{42}};print(Identity(a).Alias.Value) }`,
		`func Identity[T any](v T) T { return v }
func main(){ f:=[]func(int)int{Identity}; print(f[0](42)) }`,
		`func Identity[T any](v T) T { return v }
func main(){ f:=Identity[int];print(f(42)) }`,
		`func Identity[T any](v T) T { return v }
func Build[T any](v T) T { const N=2;type Local [N]T; a:=Local{v,v};return Identity(a)[0] }
func main(){print(Build(42))}`,
		`type Value int;type Alias=Value;func(v Alias) Get() int {return int(v)}
func Read[T interface{Get() int}](v T) int { return v.Get() };func main(){print(Read(Value(42)))}`,
		`func Each[S ~[]E,E any](values S) func(func(E) bool) { return func(yield func(E) bool){ for _,v:=range values { if !yield(v) { return } } } }
func Sum[T ~int](n T) T { var total T;for i:=range n { total+=i };return total }
func main(){ print(Sum(10)) }`,
		`func Get[M ~map[K]V,K comparable,V any](m M,k K) (V,bool) { v,ok:=m[k];return v,ok }
func Assert[T any](v any) (T,bool) { value,ok:=v.(T);return value,ok }
func main(){v,ok:=Get(map[string]int{"key":42},"key");print(v);print(ok);a,b:=Assert[int](any(42));print(a);print(b)}`,
		`func Shift[T ~int|~int64,U ~uint|~uint32](v T,n U) T { return v<<n }
func Unit[T ~uint](n T) int { return 1<<n }
func main(){print(Shift(int64(21),uint32(1)));print(Unit(uint(3)))}`,
		`func Reset[T ~[]int|~map[string]int](v T) { clear(v) }
func Remove[T ~map[string]int|~map[string]string](v T) { delete(v,"key") }
func Closed[T ~chan int|~chan string](v T) { close(v) }
func main(){ Reset([]int{1}); Remove(map[string]string{"key":"value"}) }`,
		`func Copy[S ~[]E,E any](dst,src S) int { return copy(dst,src) }
func Build[S ~[]E,E any](v E) S { s:=make(S,0,2);return append(s,v) }
func main(){ s:=Build[[]int](42);print(Copy(s,s)) }`,
		`func len(v int) int { return v };func Call[T ~int](v T) T { return T(len(int(v))) };func main(){print(Call(42))}`,
		`func Identity[T any](v T) T { return v }
type F func(int) int
func main(){ var id F=Identity; f:=func(g F) int { return g(42) }; print(f(Identity));print(id(42)) }`,
		`func Identity[T any](v T) T { return v }
type Box[T any] struct { Value T }
type Inner[T any] struct { Box[T] }
type Outer[T any] struct { *Inner[T] }
func main(){ o:=Outer[int]{&Inner[int]{Box[int]{42}}}; print(Identity(o.Value)); print(o.Box.Value) }`,
		`func Identity[T any](v T) T { return v }
func Wrap[T any](v T) func()T { return func() T { type Box struct { Value T }; b:=Box{v}; return Identity(b).Value } }
func main(){ print(Wrap(42)()) }`,
		`func Clamp[T ~int|~float64](v,lo,hi T) T { return min(max(v,lo),hi) }
func Small[T ~int8]() T { return min(127,128) }
func main(){ print(Clamp(42,0,100));print(Small[int8]()) }`,
		`func Slice[S ~[]E,E any](v S) []E { return v }
func Defined[S ~[]E,E any](v []E) S { return v }
type Ints []int
func main(){ print(len(Slice(Ints{1,2}))+len(Defined[Ints]([]int{3}))) }`,
		`func Bytes[S ~[]byte]() S { return []byte("yes") }; func main(){print(len(Bytes[[]byte]()))}`,
		`func Identity[T any](v T) T { return v }
func Wrap[T any](v T) func(T)T { return func(value T) T { return Identity(value) } }
func main(){print(Wrap(42)(7))}`,
		`func Identity[T any](v T) T { return v }
func main(){ captured:=21; f:=func(v int) int { return Identity(v)+Identity(captured) }; print(f(21)) }`,
		`type Box[T any] struct{Value T}
func (b Box[T]) Get() T { return b.Value }
type Outer[T any] struct{Box[T]}
func Read[T interface{ Get() E },E any](v T) E { return v.Get() }
func main(){o:=Outer[int]{Box:Box[int]{42}};print(Read(o));print(o.Box.Value)}`,
		`type Box[T any] struct{Value T}
func (b *Box[T]) Get() T { return b.Value }
type Outer[T any] struct{*Box[T]}
func Read[T interface{ Get() E },E any](v T) E { return v.Get() }
func main(){print(Read(Outer[int]{&Box[int]{42}}))}`,
		`func Identity[T any](v T) T { return v }
func main(){ type Local int; print(Identity(Local(42))); { type Local string; print(Identity(Local("yes"))) }; print(Identity(Local(7))) }`,
		`func Identity[T any](v T) T { return v }
func Local[T any](v T) T { type Box struct{ Value T; Next *Box }; box:=Box{Value:v}; return Identity(box).Value }
func main(){print(Local(42));print(Local("yes"))}`,
		`func Identity[T any](v T) T { return v }
func Local[T any](v T) T { type (Box struct{ Value T }; Alias = Box); box:=Alias{Value:v}; return Identity(box).Value }
func main(){print(Local(42))}`,
		`func Identity[T any](v T) T { return v }
func Get() func(int)int { return Identity }
func Use(f func(int)int) int { return f(42) }
func main(){ var f func(int)int; f=Identity; print(Use(Identity)+Get()(42)+f(42)) }`,
		`func Pair[A,B any](a A,b B) B { return b }
func Use[T,U any](f func(T,U) U,v U) U { var z T; return f(z,v) }
func main(){ print(Use(Pair[int],42)) }`,
		`func Identity[T any](v T) T { return v }
func Use[A,B any](f func(A)A,g func(B)B,a A,b B) A { return f(a) }
func main(){ print(Use(Identity,Identity,42,"text")) }`,
		`func Identity[T any](v T) T { return v }
func Get[T any]() func(T)T { return Identity }
func main(){ print(Get[int]()(42)) }`,
		`func Clone[S ~[]E,E any](s S) S { return append(S(nil),s...) }; func main(){print(len(Clone([]int{1,2})))}`,
		`func Value[T ~int8|~float32]() T { return 120+7 }; func main(){print(Value[int8]())}`,
		`func Value[T ~int]() T { return 1.5+0.5 }; func main(){print(Value[int]())}`,
		`type A[T any] struct { Value T; Next *B[T] }; type B[T any] struct { Value T; Next *A[T] }
func Identity[T any](v T) T { return v }
func main(){ var a A[int]; var b B[int]; a.Next=&b; b.Next=&a; print(Identity(a.Next.Next.Value)) }`,
		`type Box[T any] struct{Value T}; func(b Box[_]) Len() int { return 1 }; func main(){ b:=Box[int]{}; print(b.Len()) }`,
		`type Box[T any] struct{Value T}
func(b *Box[T]) Get() T { return b.Value }
func(b Box[T]) Read() T { return b.Get() }
func Identity[T any](v T) T { return v }
func main() { b:=Box[int]{Value:42}; print(Identity((*Box[int]).Get(&b))) }`,
		`type T int
func Identity[T any](v T) T { return v }
func main() { print(Identity(T(42))) }`,
		`func Keys[K comparable,V any](m map[K]V) []K { out:=make([]K,0,len(m)); for key:=range m { out=append(out,key) }; return out }
func main() { print(len(Keys(map[string]int{"answer":42}))) }`,
		`func Map[S ~[]E,E any,U any](s S,f func(E)U) []U { out:=make([]U,0,len(s)); for _,v:=range s { out=append(out,f(v)) }; return out }
func main() { print(len(Map([]int{1,2},func(v int) string { return "yes" }))) }`,
		`func Pair[T any](v T) (T,T) { return v,v }
func Forward[T any](v T) (T,T) { return Pair(v) }
func Identity[T any](v T) T { return v }
func main() { a,b:=Forward(42); print(Identity(a)+Identity(b)) }`,
		`type Value int
func (v Value) Get() int { return int(v) }
func Read[T interface { Get() E }, E any](v T) E { return v.Get() }
func main() { print(Read(Value(42))) }`,
		`func Copy[S ~[]E, E any](s S) S { return s }
func Outer[S ~[]E, E any](s S) S { return Copy(s) }
func main() { print(len(Outer([]int{1,2}))) }`,
		`type List[T any] = []T
func First[S ~[]E, E any](s S) E { return s[0] }
func main() { var a List[int] = []int{42}; print(First(a)) }`,
		`func Identity[T any](v T) T { return v }
func main() { var f func(int) int = Identity; print(f(42)) }`,
		`func Identity[T any](v T) T { return v }
func Apply[T any](f func(T) T, v T) T { return f(v) }
func main() { print(Apply(Identity, 42)) }`,
	} {
		t.Run(source, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;\n" + source)}})
			got := PrepareGenerics(graph)
			if !got.Ok {
				t.Fatalf("%s at token %d (%s)", got.Message, got.ErrorToken, tokenString(&graph.Packages[got.ErrorPackage].Files[got.ErrorFile].File, got.ErrorToken))
			}
			if checked := CheckGraph(got.Graph); !checked.Ok {
				t.Log(string(got.Graph.Packages[checkRootPackage(t, graph)].Files[0].Src))
				t.Fatalf("concrete check: %d at token %d", checked.Error, checked.ErrorToken)
			}
		})
	}
}

func TestGenericPreparationReceiverMethods(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Box[T any] struct { Value T }
func (b Box[Element]) Get() Element { return b.Value }
func (b *Box[Element]) Set(v Element) { b.Value = v }
func Identity[T any](v T) T { return v }
func main() {
 var b Box[int]
 b.Set(42)
 print(Identity(b.Get()))
}
`)}})
	prepared := PrepareGenerics(graph)
	if !prepared.Ok {
		t.Fatal(prepared.Message)
	}
	root := checkRootPackage(t, graph)
	src := string(prepared.Graph.Packages[root].Files[0].Src)
	if checked := CheckGraph(prepared.Graph); !checked.Ok {
		t.Fatalf("concrete check: %d at %d/%d/%d\n%s", checked.Error, checked.ErrorPackage, checked.ErrorFile, checked.ErrorToken, src)
	}
}

func TestGenericPreparationCrossPackage(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
import "example.com/case/lib"
type number int
func main() { print(lib.Identity(number(42))); var b lib.Box[int]; print(b.Get()) }
`)},
		{Path: "/repo/case/lib/lib.go", Src: []byte(`package lib
type Box[T any] struct { Value T }
func (b Box[T]) Get() T { return b.Value }
func Identity[T any](v T) T { return v }
`)},
	})
	prepared := PrepareGenerics(graph)
	if !prepared.Ok {
		t.Fatal(prepared.Message)
	}
	if checked := CheckGraph(prepared.Graph); !checked.Ok {
		for _, pkg := range prepared.Graph.Packages {
			for _, f := range pkg.Files {
				t.Log(string(f.Src))
			}
		}
		t.Fatalf("concrete check: %d at %d/%d/%d", checked.Error, checked.ErrorPackage, checked.ErrorFile, checked.ErrorToken)
	}
}

func TestGenericPreparationImportedVisibility(t *testing.T) {
	for _, tc := range []struct {
		source string
		ok     bool
	}{
		{`import "example.com/case/lib"; func main(){ print(lib.hidden(42)) }`, false},
		{`import . "example.com/case/lib"; func main(){ print(Identity(42)) }`, true},
	} {
		graph := genericTestGraph(t, []load.SourceFile{
			{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main; " + tc.source)},
			{Path: "/repo/case/lib/lib.go", Src: []byte("package lib; func Identity[T any](v T) T{return v}; func hidden[T any](v T) T{return v}")},
		})
		got := PrepareGenerics(graph)
		if got.Ok != tc.ok {
			t.Fatalf("%s: %s", tc.source, got.Message)
		}
		if tc.ok {
			if checked := CheckGraph(got.Graph); !checked.Ok {
				t.Fatalf("concrete check: %d\n%s", checked.Error, got.Graph.Packages[checkRootPackage(t, graph)].Files[0].Src)
			}
		}
	}
}

func TestGenericPreparationFunctionsAndTypes(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Node[T any] struct { Value T; Next *Node[T] }
func Identity[T any](v T) T { return v }
func Double[T ~int](v T) T { return v+v }
func main() {
 v := Identity[int](21)
 var n Node[int]
 n.Value = Double(v)
 print(Identity(n.Value))
}
`)}})
	prepared := PrepareGenerics(graph)
	if !prepared.Ok {
		t.Fatalf("prepare: %s at %d/%d/%d", prepared.Message, prepared.ErrorPackage, prepared.ErrorFile, prepared.ErrorToken)
	}
	root := checkRootPackage(t, graph)
	src := string(prepared.Graph.Packages[root].Files[0].Src)
	file := &prepared.Graph.Packages[root].Files[0].File
	unlowered := file.Generics != nil
	for token := 0; token+1 < len(file.Tokens); token++ {
		name := tokenString(file, token)
		if (name == "Identity" || name == "Node") && tokCharIs(file, token+1, '[') || name == "Double" && tokCharIs(file, token+1, '(') {
			unlowered = true
		}
	}
	if unlowered {
		t.Fatalf("unlowered generics:\n%s", src)
	}
	if checked := CheckGraph(prepared.Graph); !checked.Ok {
		t.Fatalf("concrete check: %d at %d/%d/%d\n%s", checked.Error, checked.ErrorPackage, checked.ErrorFile, checked.ErrorToken, src)
	}
	if !strings.Contains(string(graph.Packages[root].Files[0].Src), "Identity[T any]") {
		t.Fatal("preparation mutated original source graph")
	}
}

func TestGenericPreparationValidatesUnusedDefinitions(t *testing.T) {
	for _, definition := range []string{
		`func Bad[T any](ch chan int){const n=len([1]int{<-ch});_=n}`,
		`func Value()int{return 0};func Bad[T any](){const n=len([1]int{Value()});_=n}`,
		`func Value()[3]int{return [3]int{}};func Bad[T any](){const n=len(Value());_=n}`,
		`func Bad[T ~[3]int](v T){const n=len(v);_=n}`,
		`func Bad[T ~int](){const n T=1;_=n}`,
		`func Bad[T any](v int){const n=v;_=n}`,
		`func Bad[T ~int](v T) { v <<= int(-1) }`,
		`func Bad[T ~int](v T) { v /= 0 }`,
		`func Bad[T any](n uint) { _ = 1.0<<n }`,
		`func Bad[T any](n uint) { x:=1.0<<n;_ = x }`,
		`func Bad[T any](n uint) { var x=1.0<<n;_ = x }`,
		`func Bad[T any](n uint) float64 { return 1<<n }`,
		`func Bad[T any](n uint) int8 { return (1<<n)+128 }`,
		`func Bad[T any](n uint) int8 { return -(128<<n) }`,
		`func Bad[T any](n uint) bool { return 1<<n == 1.0 }`,
		`func Bad[T any](n uint) int8 { return min(1<<n,128) }`,
		`func Bad[T any](n uint) { _ = string(65<<n) }`,
		`func Bad[T any](n uint) { _ = float64(1<<n) }`,
		`func Bad[T any](n uint) { _ = (1<<n)/0 }`,
		`func Bad[T any](v int) { _ = v/0 }`,
		`func Bad[T any](v T) { v,x:=1,2;_,_=v,x }`,
		`func Bad[T any]() { v:=1;v,x:="bad",2;_,_=v,x }`,
		`func Bad[T ~[]int](v T) { _ = v[2:1] }`,
		`func Bad[T ~[]int](v T,n int) { _ = v[2:n:1] }`,
		`func Bad[T ~[]int](v T,n int) { _ = v[n:2:1] }`,
		`func Bad[T ~[2]int](v T) { _ = v[:3] }`,
		`func Bad[T any](v *[2]T) { _ = v[3:] }`,
		`func Bad[T any]() { _ = "abc"[4:] }`,
		`func Bad[T any]() { _ = "abc"[:4] }`,
		`func Bad[T any]() { _ = "abc"[2:1] }`,
		`func Bad[T any](v T) { _ = [2]T{v,v}[:] }`,
		`func Bad[T any](f func()[2]T) { _ = f()[:] }`,
		`func Bad[T ~[]int](v T) { _ = v[0::2] }`,
		`func Bad[T ~[]int](v T) { _ = v[0:1:] }`,
		`func Bad[S ~[]E,E any]() { _ = make(S,2,1) }`,
		"func Bad[T ~[2]int|~[3]int](v T) { _ = v[2] }",
		"func Bad[T ~[2]int|~[]int](v T) { _ = v[2] }",
		"func Bad[T any]() { _ = 1/(float32(1)+16777217-16777216) }",
		"func Bad[T any]() { _ = 1 + true }",
		"func Bad[T any]() { _ = int(1) + int64(2) }",
		"func Bad[T any]() { _ = int8(1) + 128 }",
		"func Bad[T any]() { _ = int8(1) - 128 }",
		"func Bad[T any]() { _ = float32(1) + 1e100 }",
		"func Bad[T any]() { _ = int(1)/0 }",
		"func Bad[T any]() { _ = 1 == true }",
		"func Bad[T any]() { _ = []int{} == []int{} }",
		"func Bad[T any]() { _ = nil == nil }",
		"func Bad[T any]() { _ = 1 << -1 }",
		"func Bad[T any]() { _ = float32(1) << 1 }",
		"func Bad[T any]() { _ = !1 }",
		"func Bad[T any]() { _ = +\"x\" }",
		"func Bad[T any]() { _ = ^float64(1) }",
		"func Bad[T any]() { _ = &func(){} }",
		"func Bad[T any](v T) { _ = []T{-1:v} }",
		"func Bad[T any](v T) { _ = []T{1.5:v} }",
		"func Bad[T any](v T,n int) { _ = []T{n:v} }",
		"func Bad[T any](v T) { _ = []T{0:v,0:v} }",
		"func Bad[T any](v T) { _ = []T{1:v,0:v,v} }",
		"func Bad[T any](v T) { _ = [2]T{2:v} }",
		"func Bad[T any](v T) { _ = [2]T{v,v,v} }",
		"func Bad[T any](v T) { _ = [2]T{1:v,v} }",
		"func Bad[T any](v [2]T) { _ = v[2] }",
		"func Bad[T any](v []T) { _ = v[uint64(18446744073709551615)] }",
		"const Big int = 128;func Bad[T ~int8]() T {return T(Big)}",
		"func Bad[T ~int8]() T {return T(int(128))}",
		"func Bad[T any](v T) { switch v.(type){} }",
		"func Bad[T any](v T) { switch v {} }",
		"func Bad[T ~<-chan int](v T) { select {case v<-1:default:} }",
		"func Bad[T any](v T) { _=v[:] }",
		"func Bad[T ~string|~[]byte](v T) { _=v[0:1:2] }",
		"func Bad[T ~[]int](v T) { _=v[1.5:] }",
		"func Bad[T ~[]int](v T) { _=v[-1] }",
		"func Bad[T ~map[string]int](v T) { _=v[1] }",
		"func Bad[T any](v T) { v++ }",
		"func Bad[T ~string](v T) { v++ }",
		"func Bad[T any](v T) { v+=v }",
		"func Bad[T ~chan int](v T) { v<-\"bad\" }",
		"func Bad[T ~<-chan int](v T) { v<-1 }",
		"func Bad[T any](v T) { v<-1 }",
		"func Bad[T any]() { _=[]T{1} }",
		"func Bad[T any]() { _=struct{Value T}{Value:1} }",
		"func Bad[T any]() { _=map[T]int{} }",
		"type Box[T any] struct{};type Alias=Box[int];func(v Alias) Method(){}",
		"func _[T ~int](v T) { _=v+1 };func _[T any](v T) { _=v+1 }",
		"type Bad[T any] T",
		"func Bad[T any]() { type Local T }",
		"func Bad[T any](v T) { for range v {} }",
		"func Bad[T ~chan<- int](v T) { for range v {} }",
		"func Bad[T ~int](v T) { for v {} }",
		"func Bad[T any](v T) { _=v.(int) }",
		"func Bad[T any](v T) { _=<-v }",
		"func Bad[T ~chan<- int](v T) { _=<-v }",
		"func Bad[T ~int](v T) T { return v<<1.5 }",
		"func Bad[T any](v T) { _ = *v }",
		"func Bad[T ~map[int]int](m T) { _ = &m[0] }",
		"func Bad[T any]() { const n=1;n++ }",
		"func Bad[T any]() { const n=1;n=2 }",
		"func Bad[T any](v T) { for ;false;v++ {} }",
		"func Bad[T any](v T) { for v=v+v;false; {} }",
		"func Bad[T any](v T) { if v=v+v;true {} }",
		"func Bad[T any](v T) { v,v = v }",
		"func Bad[T ~float64](v T) T { return v<<2 }",
		"func Bad[T any]() { missing() }",
		"func Bad[T any]() { var unused T }",
		"func Consume(v int) {}; func Bad[T any](v T) { Consume(v) }",
		"func Bad[T any](v T) { f:=func(value int) {};f(v) }",
		"func Bad[T any](v T) { clear(v) }",
		"func Bad[T ~[]int|~string](v T) { clear(v) }",
		"func Bad[T ~map[string]int|~map[int]int](v T) { delete(v,0) }",
		"func Bad[T ~chan int|~<-chan int](v T) { close(v) }",
		"func Bad[T ~complex64](v T) { _=real(v) }",
		"func Bad[T ~float64](v T) { _=complex(v,v) }",
		"func Bad[T any](v T) { _=append(v,v) }",
		"func Bad[T ~[]int](v T) { _=append(v,\"bad\") }",
		"func Bad[T ~[]int](v T) { _=copy(v,[]string{}) }",
		"func Bad[T ~int]() { _=make(T) }",
		"func Bad[T ~struct{ Value int }](v T) int { return v.Value }",
		"func Bad[T any]() T {}",
		"func Bad[T any](v T, ok bool) T { if ok { return v } }",
		"func Bad[T any]() { _ = func() T {} }",
		"type Bad[T any] struct { T }",
		"type Bad[T any] struct { V T; V T }",
		"func Bad[T any](v T) T { return min(v,v) }",
		"func Bad[T ~int8]() T { return max(127,128) }",
		"func Bad[T any]() { _ = func(v T) T { return v+v } }",
		"func Bad[T any](v T) T { return v+v }",
		"func Bad[T any]() T { return 1 }",
		"func Bad[T ~int | ~string](v T) T { return v-v }",
		"func Bad[T any](v T) { var n int = v; _ = n }",
		"func Bad[T any](v T) { var n int; n = v }",
		"func Bad[T any](v T) { _ = v[0] }",
		"func Bad[T any](v T) { _ = len(v) }",
		"func Bad[T any](v T) { _ = -v }",
		"func Bad[T any]() T { return }",
		"func Bad[T ~int8]() T { return 128 }",
		"func Bad[T ~int8|~uint8]() T { return -1 }",
		"func Bad[T ~int]() T { return 1.5 }",
		"func Bad[T ~float32]() T { return 1e100 }",
		"func Bad[T ~int]() T { return 1+1i }",
		"func Bad[T ~int8]() T { return T(128) }",
		"func Bad[T any](value T) int { return int(value) }",
		"func Bad[T any](T T) {}",
		"func Bad[T any](v T,v T) {}",
		"func Bad[T any]() {}; var Bad int",
		"func Bad[T any]() {}; func Bad[T any]() {}",
		"type Bad[T any] map[T]int",
		"type Good[T comparable] struct{ Value T }; type Bad[T any] Good[T]",
		"type Bad[T any] struct { Value Bad[T] }",
		"type OnlyNumbers interface{~int}; func Bad[T any](value OnlyNumbers) {}",
		"func Bad[T any]() { Bad[[]T]() }",
		"func Bad[T any](v T) { Bad([]T{v}) }",
		"func Bad[T any](v T) { var f func([]T) = Bad; _ = f }",
		"type Box[T any] struct{}; func(b Box[T]) Grow(){ var other Box[[]T]; other.Grow() }",
		"func F[T any]() { G[[]T]() }; func G[U any]() { F[U]() }",
	} {
		t.Run(definition, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main; " + definition + "; func main() {}")}})
			if got := PrepareGenerics(graph); got.Ok {
				t.Fatal("accepted invalid unused definition")
			}
		})
	}
}

func TestGenericPreparationAllowsFiniteRecursiveInstantiation(t *testing.T) {
	graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
func F[T any](n int) { if n>0 { F[[]int](n-1) } }
func main() { F[int](3) }
`)}})
	got := PrepareGenerics(graph)
	if !got.Ok {
		t.Fatal(got.Message)
	}
	if checked := CheckGraph(got.Graph); !checked.Ok {
		t.Fatalf("concrete check: %d", checked.Error)
	}
}
