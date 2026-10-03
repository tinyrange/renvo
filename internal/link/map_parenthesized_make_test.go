package link

import (
	"strings"
	"testing"

	"renvo.dev/internal/load"
)

func TestParenthesizedMapMakeLowersConstruction(t *testing.T) {
	for _, tc := range []struct{ declaration, typ string }{
		{"", "map[string]int"},
		{"type M map[string]int", "M"},
		{"type M (map[string]int)", "M"},
		{"type M = (map[string]int)", "M"},
	} {
		t.Run(tc.declaration, func(t *testing.T) {
			built := buildFromFiles(t, []load.SourceFile{
				{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + tc.declaration + ";func main(){m:=make(((" + tc.typ + ")),2);m[\"value\"]=42;if m[\"value\"]!=42{panic(1)};print(\"PASS\\n\")}\n")},
			})
			linked := LinkBuildCore(built)
			if !linked.Ok {
				t.Fatalf("link: %d", linked.Error)
			}
			text := string(linked.Program.Text)
			if strings.Contains(text, "m:=make(") || !strings.Contains(text, "m:=__renvo_map_") {
				t.Fatalf("map construction not lowered:\n%s", text)
			}
			if strings.Contains(text, "type M (") || strings.Contains(text, "type M = (") {
				t.Fatalf("parenthesized declaration reached the backend:\n%s", text)
			}
		})
	}
}

func TestParenthesizedMapResultsAreFunctionBodies(t *testing.T) {
	for name, source := range map[string]string{
		"declaration": `func value() (map[string]int){return make(map[string]int)};func main(){m:=value();m["x"]=42}`,
		"literal":     `func main(){value:=func() ((map[string]int)){return map[string]int{"x":42}};m:=value();_=m["x"]}`,
	} {
		t.Run(name, func(t *testing.T) {
			linked := LinkBuildCore(buildFromFiles(t, []load.SourceFile{
				{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + source)},
			}))
			if !linked.Ok {
				t.Fatalf("map result was mistaken for a literal: %d", linked.Error)
			}
		})
	}
}
