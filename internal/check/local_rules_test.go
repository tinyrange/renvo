package check

import (
	"renvo.dev/internal/load"
	"testing"
)

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

func TestLocalRules(t *testing.T) {
	for _, tc := range []struct {
		source string
		code   int
	}{
		{`func main(){var x uint8=256;_=x}`, CheckErrType},
		{`func main(){x:=1;x:=2;_=x}`, CheckErrScope},
		{`func main(){var x int="oops";_=x}`, CheckErrType},
		{`func main(){s:="abc";s[0]=120}`, CheckErrAssignTarget},
		{`func main(){a:=[]int{1};b:=[]int{1};_=a==b}`, CheckErrOperand},
		{`type string int; func main(){var x string=1;_=x}`, CheckOK},
		{`func main(){type uint8 int; var x uint8=256;_=x}`, CheckOK},
		{`func main(){var true=1;var x int=true;_=x}`, CheckOK},
		{`func main(){a:=[]int{1};_=a==nil}`, CheckOK},
		{`func main(){a:=[1]int{1};b:=[1]int{1};_=a==b}`, CheckOK},
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + tc.source)}})
		result := CheckGraphCore(graph)
		if result.Ok != (tc.code == CheckOK) || !result.Ok && result.Error != tc.code {
			t.Fatalf("%s: ok=%v error=%d want=%d", tc.source, result.Ok, result.Error, tc.code)
		}
	}
}
