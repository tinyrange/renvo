package check

import (
	"renvo.dev/internal/load"
	"testing"
)

func TestGenericGraphRuntimePrimitiveTypes(t *testing.T) {
	for _, body := range []string{
		`fd:=open("file",0);var n int=Id(fd);n=close(fd);_=n`,
		`fd:=open("file",0);buf:=make([]byte,8);var n int=Id(read(fd,buf,-1));n=write(fd,buf,-1);n=chmod(fd,0644);n=close(fd);_=n`,
		`var n int=Id(write(1,"value",-1));_=n`,
		`close:=func()string{return "value"};var text string=Id(close());_=text`,
		`open:=func()string{return "value"};var text string=Id(open());_=text`,
		`var ch chan int;close(ch)`,
	} {
		t.Run(body, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;func Id[T any](v T)T{return v};func main(){" + body + "}")}})
			if prepared := PrepareGenerics(graph); !prepared.Ok {
				t.Fatalf("prepare: %s", prepared.Message)
			}
		})
	}
}

func TestGenericRuntimePrimitiveArguments(t *testing.T) {
	for _, source := range []string{
		`func Bad[T ~int](v T){close(v)}`,
		`func Bad[T ~int|~chan int](v T){close(v)}`,
		`func Bad[T ~<-chan int](v T){close(v)}`,
		`func Bad[T any](v T){close(v)}`,
		`func Bad[T any](v T){_=open(v,0)}`,
		`func Bad[T any](){_=open("file")}`,
		`func Bad[T any](){_=read(0,"immutable",-1)}`,
		`func Bad[T any](){_=write("fd",[]byte{},-1)}`,
		`func Bad[T any](){_=chmod(0,true)}`,
		`func Bad[T any](){var ch <-chan int;close(ch)}`,
	} {
		t.Run(source, func(t *testing.T) {
			graph := genericTestGraph(t, []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: []byte("package main;" + source + ";func main(){}")}})
			if prepared := PrepareGenerics(graph); prepared.Ok {
				t.Fatal("invalid runtime primitive or channel close accepted")
			}
		})
	}
}
