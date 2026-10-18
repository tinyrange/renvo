package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestIndexCollectionExcludesFunctionParameterTypes(t *testing.T) {
	for _, body := range []string{
		`var v [16]byte; _ = v; f := func(v [16]byte) [16]byte { return v }; _ = f`,
		`var v [16]byte; _ = v; f := func(v *[16]byte) (out [16]byte) { return *v }; _ = f`,
	} {
		files := []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main; func main(){" + body + "}")}}
		program := CheckGraphCore(checkTestGraph(t, files))
		if !program.Ok {
			t.Fatalf("signature interpreted as index: %#v", program)
		}
	}
	files := []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main; func main(){ var v [16]byte; f:=func(p [16]byte){ _ = v[16] }; _=f }`)}}
	program := CheckGraphCore(checkTestGraph(t, files))
	if program.Ok || program.Error != CheckErrArrayIndex {
		t.Fatalf("lost body index check: %#v", program)
	}
}
