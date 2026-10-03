package build

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestGenericBuiltinIdentitySurvivesBuildModes(t *testing.T) {
	files := []load.SourceFile{{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")}, {Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main
type int string
func Id[T any](v T)T{return v}
func main(){var v any=Id(42);_=v}
`)}}
	for _, mode := range []string{"direct", "session", "transient", "cached"} {
		t.Run(mode, func(t *testing.T) {
			for pass := 0; pass < 2; pass++ {
				workspace := load.LoadWorkspace("/repo/case", "/std", "./cmd/app", files)
				if !workspace.Ok {
					t.Fatalf("workspace error=%d", workspace.Error)
				}
				graph := workspace.Graph
				var result Result
				switch mode {
				case "direct":
					result = BuildObjectPrograms(graph)
				case "transient":
					result = BuildProgramsTransient(graph)
				case "cached":
					result = BuildProgramsTransientCached(graph)
				default:
					session := BeginProgramsSession(graph, false, false)
					for !session.Step() {
					}
					result = session.Result()
				}
				if !result.Ok {
					t.Fatalf("pass %d: build error=%d message=%s", pass, result.Error, result.ErrorMessage)
				}
				if result.Root != 0 || len(result.Units) != 2 {
					t.Fatalf("root/units=%d/%d", result.Root, len(result.Units))
				}
				if result.Units[result.Root].ImportPath != graph.Root {
					t.Fatal("root index lost")
				}
			}
		})
	}
}
