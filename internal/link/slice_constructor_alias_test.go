package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestSliceConstructorImportAlias(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/value/value.go", Src: []byte("package value\ntype Value struct { n int }\nfunc Int(n int) Value { return Value{n} }\nfunc Native(s string) Value { return Value{len(s)} }\nfunc (v Value) Number() int { return v.n }\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport vm \"example.com/case/value\"\nfunc use(v []vm.Value, w []vm.Value) int { return v[0].Number()+v[1].Number()+w[0].Number() }\nfunc main(){ n := use([]vm.Value{vm.Int(2), vm.Native(\"abc\")}, []vm.Value{vm.Int(4)}); println(n) }\n")},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal("link failed")
	}
	if bytes.Contains(linked.Program.Text, []byte("vm.")) {
		t.Fatalf("unresolved alias: %s", linked.Program.Text)
	}
}
