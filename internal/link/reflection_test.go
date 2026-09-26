package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"testing"
)

func TestReflectionOptInSurvivesLinkModes(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/std/reflect/reflect.go", Src: []byte(`package reflect
type Field struct { Name, Tag string }
type Struct struct { Name string; Fields []Field }
func Describe(value any) (Struct,bool) { return Struct{},false }
func FieldValue(value any,index int) (any,bool) { return nil,false }
func SetField(value any,index int,replacement any) bool { return false }
`)},
		{Path: "/repo/case/model/model.go", Src: []byte(`package model
//renvo:reflect
type Record struct { Public int "json:\"public\""; hidden string }
type Plain struct { Public int }
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
import "reflect"
import "example.com/case/model"
func main() { meta,ok:=reflect.Describe(model.Record{}); if !ok || meta.Name!="Record" { panic("metadata") } }
`)},
	}
	for _, mode := range []string{"normal", "incremental", "transient"} {
		t.Run(mode, func(t *testing.T) {
			input := buildFromFiles(t, files)
			var linked Result
			if mode == "incremental" {
				linked = LinkBuildCoreIncremental(input)
			} else if mode == "transient" {
				linked = LinkBuildCoreTransient(input)
			} else {
				linked = LinkBuildCore(input)
			}
			if !linked.Ok {
				t.Fatalf("link: %d", linked.Error)
			}
			if !bytes.Contains(linked.Data, []byte("Tag:\"json:\\\"public\\\"\"")) || bytes.Contains(linked.Data, []byte("Name:\"hidden\"")) ||
				!bytes.Contains(linked.Data, []byte("record.Public")) || bytes.Contains(linked.Data, []byte("record.hidden")) {
				t.Fatalf("incorrect descriptor/accessors in %s linked unit", mode)
			}
		})
	}
}
