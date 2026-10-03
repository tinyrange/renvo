package link

import (
	"bytes"
	"renvo.dev/internal/load"
	"renvo.dev/internal/unit"
	"testing"
)

func TestGenericReflectionNamesSurviveLinkAndCacheModes(t *testing.T) {
	files := []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")},
		{Path: "/std/reflect/reflect.go", Src: []byte(`package reflect
type Field struct{Name,Tag string}
type Struct struct{Name string;Fields []Field}
func Describe(value any)(Struct,bool){return Struct{},false}
func FieldValue(value any,index int)(any,bool){return nil,false}
func SetField(value any,index int,replacement any)bool{return false}
`)},
		{Path: "/repo/case/model/model.go", Src: []byte(`package model
type hidden[T any] struct{Value T}
type Public[T any] struct{Value T}
type Alias[T any] = Public[T]
//renvo:reflect
type Box[T any] struct{ Public[T]; Alias[T]; hidden[T]; Item T "json:\"item\"" }
func New[T any](value T) Box[T]{return Box[T]{Public:Public[T]{value},Alias:Alias[T]{value},hidden:hidden[T]{value},Item:value}}
`)},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
import "reflect"
import "example.com/case/model"
func main(){box:=model.New(42);_=box.Public.Value;_=box.Alias.Value;_,_=reflect.Describe(box);_,_=reflect.Describe(model.New("hello"))}
`)},
	}
	for _, mode := range []string{"normal", "incremental", "transient", "incremental_transient", "cache"} {
		t.Run(mode, func(t *testing.T) {
			input := buildFromFiles(t, files)
			if mode == "cache" {
				for i := range input.Units {
					data, ok := unit.MarshalFrontendCache(input.Units[i].Program)
					if !ok {
						t.Fatal("cache encoding")
					}
					program, ok := unit.UnmarshalFrontendCache(data)
					if !ok {
						t.Fatal("cache decoding")
					}
					input.Units[i].Program = program
				}
			}
			var linked Result
			switch mode {
			case "incremental_transient":
				session := BeginPackageSession(input, true)
				for !session.Step() {
				}
				linked = session.Result()
			case "incremental":
				linked = LinkBuildCoreIncremental(input)
			case "transient":
				linked = LinkBuildCoreTransient(input)
			default:
				linked = LinkBuildCore(input)
			}
			if !linked.Ok {
				t.Fatalf("link: %d", linked.Error)
			}
			for _, name := range []string{"Box[int]", "Box[string]", "Public", "Alias", "Item"} {
				if !bytes.Contains(linked.Data, []byte("Name:\""+name+"\"")) {
					t.Fatalf("missing semantic name %q in %s", name, mode)
				}
			}
			if bytes.Contains(linked.Data, []byte("Name:\"hidden\"")) || bytes.Contains(linked.Data, []byte("Name:\"RenvoGeneric")) {
				t.Fatalf("private or generated field leaked in %s", mode)
			}
		})
	}
}
