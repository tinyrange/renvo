package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestNewExpressionLanguageVersion(t *testing.T) {
	for _, tc := range []struct {
		version, header, source string
		want                    int
	}{
		{"1.22", "", `func main(){_=new(1)}`, CheckErrNewVersion},
		{"", "", `func main(){_=new("x")}`, CheckErrNewVersion},
		{"1.25.5", "", `func main(){_=new((true))}`, CheckErrNewVersion},
		{"1.22", "", `type N int;func f(n N){_=new(n)}`, CheckErrNewVersion},
		{"1.22", "", `func main(){_=new(int(1))}`, CheckErrNewVersion},
		{"1.22", "", `const n=1;func main(){_=new(n)}`, CheckErrNewVersion},
		{"1.26", "", `func main(){_=new(1)}`, CheckOK},
		{"1.26rc1", "", `func main(){_=new("x")}`, CheckOK},
		{"1.22", "//go:build go1.26\n\n", `func main(){_=new(1)}`, CheckOK},
		{"1.26", "//go:build go1.22\n\n", `func main(){_=new(1)}`, CheckErrNewVersion},
		{"1.22", "", `type N int;func main(){_=new(N);_=new(int);_=new(struct{});_=new([]int);_=new(*int)}`, CheckOK},
		{"1.22", "", `func main(){type N int;_=new(N)}`, CheckOK},
		{"1.22", "", `func new(n int)int{return n};func main(){_=new(1)}`, CheckOK},
		{"1.22", "", `func main(){new:=func(n int)int{return n};_=new(1)}`, CheckOK},
		{"1.26", "", `func main(){_=new()}`, CheckErrBuiltinArity},
		{"1.22", "", `func main(){_=new(int,int)}`, CheckErrBuiltinArity},
	} {
		module := load.Module{Root: "/repo/case", Path: "example.com/case", GoVersion: tc.version, Ok: true}
		files := []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte(tc.header + "package main\n" + tc.source)}}
		graph := load.LoadGraph(module, "/std", "/repo/case", "./cmd/app", files)
		if !graph.Ok {
			t.Fatalf("load failed: %d", graph.Error)
		}
		result := CheckGraphCore(graph)
		if result.Error != tc.want {
			t.Errorf("version %q header %q source %s: error=%d want=%d", tc.version, tc.header, tc.source, result.Error, tc.want)
		}
	}
}

func TestStandardLibraryNewUsesCompilerLanguageVersion(t *testing.T) {
	for _, version := range []string{"1.16", "1.26"} {
		for _, operand := range []string{"1", "int"} {
			module := load.Module{Root: "/repo/case", Path: "example.com/case", GoVersion: version, Ok: true}
			graph := load.LoadGraph(module, "/std", "/repo/case", ".", []load.SourceFile{
				{Path: "/repo/case/main.go", Src: []byte("package main\nimport \"versionprobe\"\nfunc main(){_=versionprobe.Value()}\n")},
				{Path: "/std/versionprobe/value.go", Src: []byte("package versionprobe\nfunc Value()*int{return new(" + operand + ")}\n")},
			})
			if !graph.Ok {
				t.Fatalf("load failed: %d", graph.Error)
			}
			result := CheckGraphCore(graph)
			want := CheckOK
			if operand == "1" {
				want = CheckErrNewVersion
			}
			if result.Error != want {
				t.Fatalf("application %s, std new(%s): %d want %d", version, operand, result.Error, want)
			}
		}
	}
}
