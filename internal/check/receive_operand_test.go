package check

import (
	"testing"

	"renvo.dev/internal/load"
)

func TestReceiveLiteralOperand(t *testing.T) {
	for _, source := range []string{
		`func main(){select{case <-1:}}`,
		`func main(){<-1}`,
		`func main(){_=<-((1))}`,
		`func main(){_=<-"text"}`,
		`func main(){_=<-'a'}`,
		`func main(){_=<-1i}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); result.Ok || result.Error != CheckErrOperand {
			t.Fatalf("%s: ok=%v error=%d", source, result.Ok, result.Error)
		}
	}
}

func TestReceiveLiteralOperandControls(t *testing.T) {
	for _, source := range []string{
		`func f(ch chan int){ch<-1;_=<-ch}`,
		`func f(ch chan string){ch<-"text";_=<-ch}`,
		`func f(ch chan int){select{case ch<-1:case <-ch:default:}}`,
		`func f(ch []chan int){_=<-ch[0]}`,
		`func f(ch chan int){_=<-(ch)}`,
		`func f(ch <-chan int){_=<-ch}`,
	} {
		graph := checkTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main\n" + source)}})
		if result := CheckGraphCore(graph); !result.Ok {
			t.Fatalf("%s: error=%d", source, result.Error)
		}
	}
}
