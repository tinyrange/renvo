package build

import (
	"fmt"
	"testing"

	"renvo.dev/internal/load"
)

// Keep dependency sources fixed while changing the destination and build mode.
// Array assignment requires the checker to retain the numeric layout's identity,
// rather than merely accepting the layout expression as some constant.
func TestGenericLayoutIdentityAcrossBuildContexts(t *testing.T) {
	packageProgramCacheUsed = nil
	packageProgramCacheData = nil
	for _, context := range []struct {
		name               string
		layout             load.TargetLayout
		object             bool
		pointer, aggregate int
	}{
		{"amd64", load.TargetLayout{WordBits: 64, PointerBits: 64}, false, 8, 16},
		{"386", load.TargetLayout{WordBits: 32, PointerBits: 32}, false, 8, 12},
		{"wasm32", load.TargetLayout{WordBits: 32, PointerBits: 32, ScalarAlign: 8}, false, 8, 16},
		{"object386", load.TargetLayout{WordBits: 32, PointerBits: 32}, true, 4, 12},
		{"amd64again", load.TargetLayout{WordBits: 64, PointerBits: 64}, false, 8, 16},
	} {
		t.Run(context.name, func(t *testing.T) {
			for _, mode := range []string{"direct", "session", "cached"} {
				for pass := 0; pass < 3; pass++ {
					pointer := context.pointer
					if pass == 2 {
						pointer++
					}
					files := []load.SourceFile{
						{Path: "/repo/case/cmd/app/main.go", Src: []byte(fmt.Sprintf(`package main
import "example.com/case/pkg/lib"
func main(){var p [%d]int=lib.Pointer[int]();var a [%d]int=lib.Aggregate[int]();_=p;_=a}
`, pointer, context.aggregate))},
						{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib
import "unsafe"
func Pointer[T any]() [unsafe.Sizeof((*T)(nil))]T {var v [unsafe.Sizeof((*T)(nil))]T;return v}
func Aggregate[T any]() [unsafe.Sizeof(struct{A byte;B uint64}{})]T {var v [unsafe.Sizeof(struct{A byte;B uint64}{})]T;return v}
`)},
						{Path: "/std/unsafe/unsafe.go", Src: []byte("package unsafe;type Pointer *byte")},
					}
					graph := load.LoadGraph(load.Module{Root: "/repo/case", Path: "example.com/case", GoVersion: "1.25", Ok: true}, "/std", "/repo/case", "./cmd/app", files)
					if !graph.Ok {
						t.Fatalf("load error=%d", graph.Error)
					}
					graph.Layout = context.layout
					var result Result
					if mode == "direct" {
						if context.object {
							result = BuildObjectPrograms(graph)
						} else {
							result = BuildProgramsTransient(graph)
						}
					} else {
						var session *ProgramSession
						if context.object {
							session = BeginObjectProgramsSession(graph, mode == "cached")
						} else {
							session = BeginProgramsSession(graph, false, mode == "cached")
						}
						for !session.Step() {
						}
						result = session.Result()
					}
					if result.Ok != (pass != 2) {
						t.Fatalf("%s pass %d: error=%d detail=%d message=%s", mode, pass, result.Error, result.ErrorDetail, result.ErrorMessage)
					}
				}
			}
		})
	}
}
