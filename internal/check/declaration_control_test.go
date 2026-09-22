package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestAuditTerminatingStatements(t *testing.T) {
	for _, body := range []string{
		"return 1",
		"if x { return 1 } else { return 2 }",
		"if x { return 1 } else if x { return 2 } else { return 3 }",
		"for {}",
		"for ;; {}",
		"for { switch x { case true: break } }",
		"switch x { case true: return 1; default: return 2 }",
		"switch x { case true: fallthrough; default: return 2 }",
		"select {}",
		"select { default: return 2 }",
		"panic(\"stop\")",
		"(panic)(\"stop\")",
		"((panic))(\"stop\")",
		"L: for { if x { continue L } }",
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc f(x bool) int {" + body + "}")}})
		result := CheckGraphCore(graph)
		if !result.Ok {
			t.Fatalf("%s: error=%d token=%d", body, result.Error, result.ErrorToken)
		}
	}
}

func TestDeclarationAndBranchRejections(t *testing.T) {
	cases := []struct {
		source string
		code   int
	}{
		{"func f(x bool) int { if x { return 1 } }", CheckErrMissingReturn},
		{"func f() int {}", CheckErrMissingReturn},
		{"func f() int { for { break } }", CheckErrMissingReturn},
		{"func init(x int) {}", CheckErrInitSignature},
		{"func init() int { return 1 }", CheckErrInitSignature},
		{"func (x int) M() {}", CheckErrMethod},
		{"type P *int; func (p P) M() {}", CheckErrMethod},
		{"func main(){ fallthrough }", CheckErrBody},
		{"func main(){ switch 1 {case 1: fallthrough} }", CheckErrBody},
		{"func main(){ L: {continue L} }", CheckErrContinue},
		{"func main(){ goto L; {L:} }", CheckErrScope},
		{"func main(){ goto L; x:=1; _=x; L: }", CheckErrScope},
	}
	for _, tc := range cases {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + tc.source)}})
		result := CheckGraphCore(graph)
		if result.Ok || result.Error != tc.code {
			t.Fatalf("%s: got ok=%v error=%d want %d", tc.source, result.Ok, result.Error, tc.code)
		}
	}
}
