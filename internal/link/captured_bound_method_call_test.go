package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestCapturedBoundMethodNestedCalls(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type Value int
type Builder struct{}
func (b *Builder) Constant(x uint64) Value{return Value(x)}
func (b *Builder) Binary(k int,a,c Value)Value{return a^c}
func lower(b *Builder)Value{
 constant:=b.Constant
 binary:=b.Binary
 not:=func(a Value)Value{return binary(6,a,constant(^uint64(0)))}
 return not(42)
}
func main(){println(lower(&Builder{}))}
`)},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal("link failed", linked.Error)
	}
	if bytes.Contains(linked.Program.Text, []byte("(*env.binary)(")) || bytes.Contains(linked.Program.Text, []byte("(*env.constant)(")) {
		var relevant []byte
		for _, line := range bytes.Split(linked.Program.Text, []byte("\n")) {
			if bytes.Contains(line, []byte("env.")) || bytes.Contains(line, []byte("constant")) || bytes.Contains(line, []byte("binary")) {
				relevant = append(relevant, line...)
				relevant = append(relevant, '\n')
			}
		}
		t.Fatalf("captured bound calls escaped function-value lowering:\n%s", relevant)
	}
}
