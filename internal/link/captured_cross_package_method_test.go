package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestCapturedCrossPackageBoundMethods(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/builder/builder.go", Src: []byte(`package builder
type Value int
type Builder struct{Offset int}
func (b *Builder) Constant(x uint64) Value{return Value(x)+Value(b.Offset)}
func (b *Builder) Binary(k int,a,c Value)Value{return a+c+Value(k)}
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
import "example.com/case/builder"
func lower(b *builder.Builder) builder.Value {
 constant:=b.Constant
 binary:=b.Binary
 and:=func(a builder.Value,m uint64) builder.Value{return binary(5,a,constant(m))}
 shifted:=func(a builder.Value) builder.Value{return binary(6,a,and(constant(0),1))}
 return shifted(42)
}
func main(){println(lower(&builder.Builder{}))}
`)},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal("link failed", linked.Error)
	}
	if bytes.Contains(linked.Program.Text, []byte("(*env.binary)(")) || bytes.Contains(linked.Program.Text, []byte("(*env.constant)(")) {
		t.Fatalf("cross-package bound methods escaped lowering:\n%s", linked.Program.Text)
	}
}
