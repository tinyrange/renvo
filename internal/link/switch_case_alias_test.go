package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestSwitchCasePackageAliases(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport \"example.com/case/other\"\nconst Set = 3\nfunc main(){ other.Run() }\n")},
		{Path: "/repo/case/other/other.go", Src: []byte(`package other
const Set = "set"
const Get = "get"
type Item struct { Set int }
func choose(s string) int {
 switch s {
 case Get, Set: return 3
 }
 return 0
}
func Run() {
 item := Item{Set: 7}
Set:
 for item.Set > 0 { break Set }
 if choose("set") != 3 { panic("case") }
}
`)},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatal("link failed")
	}
	if bytes.Contains(linked.Program.Text, []byte("case Get, Set:")) {
		t.Fatalf("case constant did not receive its package alias: %s", linked.Program.Text)
	}
	if !bytes.Contains(linked.Program.Text, []byte("Item{Set: 7}")) || !bytes.Contains(linked.Program.Text, []byte("break Set")) {
		t.Fatal("field or label was rewritten as a package constant")
	}
}
