package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestFunctionValueIndirectCallArgument(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Callback func(int) int
func identity(v int) int { return v }
func main() { use:=func(f Callback) int { return f(42) }; print(use(identity)) }
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("link: %d", linked.Error)
	}
	if !bytes.Contains(linked.Program.Text, []byte("use, Callback{kind:")) {
		t.Fatalf("indirect callback argument was not lowered:\n%s", linked.Program.Text)
	}
}

func TestFunctionValueNativeIndirectCallArgument(t *testing.T) {
	for _, binding := range []string{
		"var outer func(func(int)int)int=apply",
		"var boxed any=apply;outer:=boxed.(func(func(int)int)int)",
		"var boxed any=apply;outer,ok:=boxed.(func(func(int)int)int);_=ok",
	} {
		t.Run(binding, func(t *testing.T) {
			result := buildFromFiles(t, []load.SourceFile{
				{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;func apply(f func(int)int)int{return f(42)};func visit(v int)int{return v};func main(){" + binding + ";print(outer(visit));_=func(v int)int{return v}}")},
			})
			linked := LinkBuildCore(result)
			if !linked.Ok {
				t.Fatalf("link: %d", linked.Error)
			}
			if bytes.Contains(linked.Program.Text, []byte("outer(visit)")) || !bytes.Contains(linked.Program.Text, []byte("return visit(")) {
				t.Fatalf("native callback argument was not lowered:\n%s", linked.Program.Text)
			}
		})
	}
}
