package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestLinkKeepsMethodNamesWhenPackageSymbolsCollide(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/thing/thing.go", Src: []byte("package thing\n\ntype Timer struct{ stopped bool }\n\nfunc New() *Timer { return &Timer{} }\n\nfunc (t *Timer) Stop() bool {\n\tif t.stopped {\n\t\treturn false\n\t}\n\tt.stopped = true\n\treturn true\n}\n")},
		{Path: "/repo/case/waker/waker.go", Src: []byte("package waker\n\ntype Timer uint64\n\nfunc New() Timer { return 0 }\n\nfunc (t Timer) Stop() bool { return true }\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n\nimport \"example.com/case/thing\"\nimport \"example.com/case/waker\"\n\nfunc main() {\n\ttimer := thing.New()\n\tfirst := timer.Stop()\n\tsecond := timer.Stop()\n\tw := waker.New()\n\twakeStop := w.Stop()\n\tif !first || second || !wakeStop {\n\t\tprint(\"FAIL\")\n\t\treturn\n\t}\n\tprint(\"PASS\")\n}\n")},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("link failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	text := linked.Program.Text
	if bytes.Contains(text, []byte("renvop")) && bytes.Contains(text, []byte("_Stop")) {
		t.Fatalf("method declaration was renamed, orphaning call sites:\n%s", text)
	}
	if !bytes.Contains(text, []byte("timer.Stop()")) {
		t.Fatalf("method call site was rewritten:\n%s", text)
	}
}
