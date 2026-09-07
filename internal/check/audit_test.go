package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestAuditRejections(t *testing.T) {
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
		{"func f(x ...int) {}; func main() { f(1, []int{2}...) }", CheckErrCallArity},
		{"var m map[[]int]int", CheckErrMapKey},
		{"type S struct { _ []int }; var m map[S]int", CheckErrMapKey},
		{"var m map[[2]func()]int", CheckErrMapKey},
		{"var m map[map[int]int]int", CheckErrMapKey},
		{"func main(){ fallthrough }", CheckErrBody},
		{"func main(){ switch 1 {case 1: fallthrough} }", CheckErrBody},
		{"func main(){ L: {continue L} }", CheckErrContinue},
		{"func main(){ goto L; {L:} }", CheckErrScope},
		{"func main(){ goto L; x:=1; _=x; L: }", CheckErrScope},
		{"func f(x,y int) {}; func main(){f(1)}", CheckErrCallArity},
		{"func f(x int,y int) {}; func main(){f(1)}", CheckErrCallArity},
		{"func main(){var x uint8=256; _=x}", CheckErrType},
		{"func main(){x:=1;x:=2;_=x}", CheckErrScope},
		{"func main(){var x int=\"oops\";_=x}", CheckErrType},
		{"func main(){s:=\"abc\";s[0]=120}", CheckErrAssignTarget},
		{"func main(){a:=[]int{1};b:=[]int{1};_=a==b}", CheckErrOperand},
	}
	for _, tc := range cases {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + tc.source)}})
		result := CheckGraphCore(graph)
		if result.Ok || result.Error != tc.code {
			t.Fatalf("%s: got ok=%v error=%d want %d", tc.source, result.Ok, result.Error, tc.code)
		}
	}
}

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
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nfunc f(x bool) int {" + body + "}")}})
		result := CheckGraphCore(graph)
		if !result.Ok {
			t.Fatalf("%s: error=%d token=%d", body, result.Error, result.ErrorToken)
		}
	}
}

func TestAuditIndependentCaseScopes(t *testing.T) {
	for _, source := range []string{
		"func main(){switch 1 {case 1: x:=1;_=x; case 2: x:=2;_=x; default: x:=3;_=x}}",
		"func f(ch chan int){select {case <-ch: x:=1;_=x; default: x:=2;_=x}}",
		"func main(){switch 1 {case 1: switch 2 {case 2: x:=1;_=x; default: x:=2;_=x}; x:=3;_=x; default: x:=4;_=x}}",
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		result := CheckGraphCore(graph)
		if !result.Ok {
			t.Fatalf("%s: error=%d token=%d", source, result.Error, result.ErrorToken)
		}
	}
}

func TestAuditImportedVisibility(t *testing.T) {
	for _, expression := range []string{"lib.hidden", "lib.S{hidden: 1}", "lib.S{1, 2}"} {
		graph := checkTestGraph(t, []load.SourceFile{
			{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport \"example.com/case/lib\"\nfunc main() { _ = " + expression + " }")},
			{Path: "/repo/case/lib/lib.go", Src: []byte("package lib\nvar hidden = 2\ntype S struct { hidden int; X int }")},
		})
		result := CheckGraphCore(graph)
		if result.Ok || result.Error != CheckErrUndefined {
			t.Fatalf("%s: accepted or wrong error %d", expression, result.Error)
		}
	}
}

func TestUnicodeImportedVisibility(t *testing.T) {
	for _, name := range []string{"Ω", "π", "世界"} {
		graph := checkTestGraph(t, []load.SourceFile{
			{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\nimport \"example.com/case/lib\"\nfunc main(){_=lib." + name + "()}")},
			{Path: "/repo/case/lib/lib.go", Src: []byte("package lib\nfunc Ω() int{return 1};func π() int{return 2};func 世界() int{return 3}")},
		})
		result := CheckGraphCore(graph)
		if result.Ok != (name == "Ω") {
			t.Fatalf("%s: ok=%v error=%d", name, result.Ok, result.Error)
		}
	}
}
