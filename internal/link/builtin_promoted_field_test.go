package link

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestClearNestedArrayPromotedAliasField(t *testing.T) {
	built := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
 type Base struct { Pages [3]int }
 type Alias=Base
 type Context struct { Alias }
 type Machine struct {native [2][2]Context}
 func reset(m *Machine,i,j int){clear(m.native[i][j].Pages[:])}
 func main(){var m Machine;reset(&m,0,1)}
`)},
	})
	linked := LinkBuildCore(built)
	if !linked.Ok {
		t.Fatalf("promoted clear field did not link: %d", linked.Error)
	}
}
