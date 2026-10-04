package link

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	wireunit "renvo.dev/backend/unit"
	"renvo.dev/internal/load"
	"renvo.dev/internal/unit"
)

func TestIntrinsicAliasesPreserveAuthoredBindings(t *testing.T) {
	for _, tc := range []struct{ name, declarations, body string }{
		{"local_value", "", `renvo_runtime_FmtPrintln:=17;fmt.Println("PASS");return Id(renvo_runtime_FmtPrintln)`},
		{"local_callback", "", `renvo_runtime_FmtPrintln:=func()int{return 17};fmt.Println("PASS");return Id(renvo_runtime_FmtPrintln())`},
		{"parameter", `func Run(renvo_runtime_FmtPrintln func()int)int{fmt.Println("PASS");return renvo_runtime_FmtPrintln()}`, `return Run(func()int{return 17})`},
		{"result", `func Run()(renvo_runtime_FmtPrintln int){fmt.Println("PASS");return 17}`, `return Run()`},
		{"global_function", `func renvo_runtime_FmtPrintln()int{return 17}`, `fmt.Println("PASS");return Id(renvo_runtime_FmtPrintln)()`},
		{"global_type", `type renvo_runtime_FmtPrintln int`, `fmt.Println("PASS");return int(Id(renvo_runtime_FmtPrintln(17)))`},
		{"suffix", `func renvo_runtime_FmtPrintln_()int{return 17}`, `renvo_runtime_FmtPrintln:=19;renvo_runtime_FmtPrintln__:=23;fmt.Println("PASS");return renvo_runtime_FmtPrintln+renvo_runtime_FmtPrintln__+renvo_runtime_FmtPrintln_()`},
		{"field", `type Value struct{renvo_runtime_FmtPrintln int}`, `v:=Value{renvo_runtime_FmtPrintln:17};fmt.Println("PASS");return Id(v.renvo_runtime_FmtPrintln)`},
		{"syscall_function", `func renvo_runtime_Syscall()int{return 17}`, `fmt.Println("PASS");return Id(renvo_runtime_Syscall)()`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := []load.SourceFile{
				{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\ngo 1.25\n")},
				{Path: "/std/fmt/fmt.go", Src: []byte(`package fmt;func Println(values ...interface{})(int,error){return 0,nil}`)},
				{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main;import "fmt";func Id[T any](v T)T{return v};` + tc.declarations + `;func appMain()int{` + tc.body + `}`)},
			}
			for _, mode := range []string{"normal", "incremental", "transient", "incremental_transient", "cache"} {
				t.Run(mode, func(t *testing.T) {
					input := buildFromFiles(t, files)
					if mode == "cache" {
						for i := range input.Units {
							data, ok := unit.MarshalFrontendCache(input.Units[i].Program)
							if !ok {
								t.Fatal("encode frontend cache")
							}
							program, ok := unit.UnmarshalFrontendCache(data)
							if !ok {
								t.Fatal("decode frontend cache")
							}
							input.Units[i].Program = program
						}
					}
					var linked Result
					switch mode {
					case "incremental":
						linked = LinkBuildCoreIncremental(input)
					case "transient":
						linked = LinkBuildCoreTransient(input)
					case "incremental_transient":
						session := BeginPackageSession(input, true)
						for !session.Step() {
						}
						linked = session.Result()
					default:
						linked = LinkBuildCore(input)
					}
					if !linked.Ok {
						t.Fatalf("link: %d", linked.Error)
					}
					decoded, err := wireunit.Unmarshal(linked.Data)
					if err != nil {
						t.Fatal(err)
					}
					if tc.name != "syscall_function" && !bytes.Contains(decoded.Text, []byte("//renvo:intrinsic renvo_runtime_FmtPrintln\n")) {
						t.Fatal("renamed intrinsic lost its declaration marker")
					}
					source := packageAliasGoSource(decoded)
					set := token.NewFileSet()
					file, err := parser.ParseFile(set, "linked.go", source, 0)
					if err != nil {
						t.Fatalf("parse: %v\n%s", err, source)
					}
					if _, err = (&types.Config{}).Check("example.com/linked", set, []*ast.File{file}, nil); err != nil {
						t.Fatalf("check: %v\n%s", err, source)
					}
				})
			}
		})
	}
}
