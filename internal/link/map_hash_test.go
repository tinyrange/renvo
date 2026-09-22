package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestMapHashNamedKeys(t *testing.T) {
	for _, key := range []string{"int64", "string"} {
		t.Run(key, func(t *testing.T) {
			built := buildFromFiles(t, []load.SourceFile{
				{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\ntype Base " + key + "\ntype Key = Base\nfunc main() { m := make(map[Key]int); println(len(m)) }\n")},
			})
			linked := LinkBuildCore(built)
			if !linked.Ok {
				t.Fatalf("link failed: %d", linked.Error)
			}
			if !bytes.Contains(linked.Program.Text, []byte("buckets []int")) {
				t.Fatalf("named %s map still uses linear lookup", key)
			}
		})
	}
}
