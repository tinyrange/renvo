package link

import (
	"bytes"
	"testing"

	"renvo.dev/internal/load"
)

func TestLinkLowersChannelsAndGoroutinesBeforeCoreUnit(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type renvo_runtime_Pointer uintptr
func renvo_runtime_ChanCreate(size uintptr, capacity int) uintptr { return 1 }
func renvo_runtime_ChanSend(channel uintptr, value renvo_runtime_Pointer) {}
func renvo_runtime_ChanReceive(channel uintptr, value renvo_runtime_Pointer) bool { return true }
func renvo_runtime_ChanClose(channel uintptr) {}
func renvo_runtime_ChanLen(channel uintptr) int { return 0 }
func renvo_runtime_ChanCap(channel uintptr) int { return 0 }
func renvo_runtime_Spawn(entry uintptr, context uintptr) {}
func renvo_runtime_Call(entry uintptr, context uintptr) { panic("invalid") }

func consume(value int) {}
func closeBoth(first chan int, second chan int) {
	close(first)
	marker := 1
	_ = marker
	close(second)
}

func main() {
	values := make(chan int, 2)
	values <- 7
	value, open := <-values
	for item := range values { print(item) }
	go consume(value)
	if open { print(len(values), cap(values)); close(values) }
	closeBoth(make(chan int), make(chan int))
}
`)},
	})
	siteCount := 0
	for i := 0; i < len(result.Units); i++ {
		siteCount += len(result.Units[i].Program.ConcurrencySites)
	}
	if siteCount == 0 {
		t.Fatal("checked package omitted concurrency-site metadata")
	}
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("link failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	text := linked.Program.Text
	for _, forbidden := range [][]byte{[]byte("make(chan"), []byte("values <-"), []byte("<-values"), []byte("range values"), []byte("go consume"), []byte("close(second)")} {
		if bytes.Contains(text, forbidden) {
			t.Fatalf("linked program retained concurrency syntax %q:\n%s", forbidden, text)
		}
	}
	for _, required := range [][]byte{[]byte("renvo_runtime_ChanCreate"), []byte("renvo_runtime_ChanSend"), []byte("renvo_runtime_ChanReceive"), []byte("renvo_runtime_Spawn"), []byte("switch int(entry)")} {
		if !bytes.Contains(text, required) {
			t.Fatalf("linked program is missing %q:\n%s", required, text)
		}
	}
	incremental := LinkBuildCoreIncremental(result)
	if !incremental.Ok || !bytes.Equal(incremental.Data, linked.Data) {
		t.Fatal("cached package artifacts changed concurrency lowering")
	}
	if len(linked.Program.ConcurrencySites) != 0 || len(incremental.Program.ConcurrencySites) != 0 {
		t.Fatal("link-only concurrency metadata crossed into the lowered program")
	}
}

func TestLinkLowersSelectToOneHandlerDecision(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type renvo_runtime_Channel uintptr
type renvo_runtime_Pointer uintptr
type renvo_runtime_ChanSelectValue struct { Channel renvo_runtime_Channel; Value renvo_runtime_Pointer; ReceiveOK renvo_runtime_Pointer; Direction int }
const renvo_runtime_SelectReceive = 0
const renvo_runtime_SelectSend = 1
func renvo_runtime_ChanSelect(cases []renvo_runtime_ChanSelectValue, hasDefault bool) int { return -1 }
func renvo_runtime_ChanCreate(size uintptr, capacity int) uintptr { return 1 }

func main() {
	values := make(chan int, 1)
	select {
	case value, open := <-values:
		print(value, open)
	case values <- 7:
		print("sent")
	default:
		print("default")
	}
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("link failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	text := linked.Program.Text
	if bytes.Contains(text, []byte("select {")) || bytes.Contains(text, []byte("case values <-")) || bytes.Contains(text, []byte("<-values")) {
		t.Fatalf("linked program retained select syntax:\n%s", text)
	}
	if bytes.Count(text, []byte("result.index = renvo_runtime_ChanSelect(cases")) != 1 {
		t.Fatalf("select did not lower to one handler decision:\n%s", text)
	}
}

func TestLinkStagesChannelOperationsAndDynamicGoCalls(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

type namedChannel chan int
type worker struct{}

type renvo_runtime_Pointer uintptr
func renvo_runtime_ChanCreate(size uintptr, capacity int) uintptr { return 1 }
func renvo_runtime_ChanSend(channel uintptr, value renvo_runtime_Pointer) {}
func renvo_runtime_Spawn(entry uintptr, context uintptr) {}
func renvo_runtime_Call(entry uintptr, context uintptr) { panic("invalid") }

func channelValue() namedChannel { return nil }
func argument() int { return 7 }
func consume(value int) {}
func (worker) consume(value int) {}

func main() {
	values := make(namedChannel, 1)
	if values == nil || nil != values { print("nil") }
	channelValue() <- argument()
	call := consume
	go call(argument())
	item := worker{}
	go item.consume(argument())
	go func(value int) { consume(value) }(argument())
}
`)},
	})
	linked := LinkBuildCore(result)
	if !linked.Ok {
		t.Fatalf("link failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	text := linked.Program.Text
	for _, forbidden := range [][]byte{[]byte("make(namedChannel"), []byte("== nil"), []byte("nil !="), []byte("go call"), []byte("go item.consume"), []byte("go func")} {
		if bytes.Contains(text, forbidden) {
			t.Fatalf("linked program retained %q:\n%s", forbidden, text)
		}
	}
	for _, required := range [][]byte{[]byte("namedChannel(renvo_runtime_ChanCreate"), []byte(":= channelValue(); var __renvo_chan_send_"), []byte("consume((*__renvo_go_context_0)(context).value0)"), []byte("receiver0: item"), []byte("renvo_runtime_Spawn")} {
		if !bytes.Contains(text, required) {
			t.Fatalf("linked program is missing %q:\n%s", required, text)
		}
	}
}

func FuzzConcurrencyLowering(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7})
	f.Add([]byte{7, 7, 3, 5, 1, 6, 2, 0, 4})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		if len(data) > 64 {
			data = data[:64]
		}
		var source bytes.Buffer
		source.WriteString(`package main

type renvo_runtime_Channel uintptr
type renvo_runtime_Pointer uintptr
type renvo_runtime_ChanSelectValue struct { Channel renvo_runtime_Channel; Value renvo_runtime_Pointer; ReceiveOK renvo_runtime_Pointer; Direction int }
const renvo_runtime_SelectReceive = 0
const renvo_runtime_SelectSend = 1
func renvo_runtime_ChanCreate(size uintptr, capacity int) uintptr { return 1 }
func renvo_runtime_ChanSend(channel uintptr, value renvo_runtime_Pointer) {}
func renvo_runtime_ChanReceive(channel uintptr, value renvo_runtime_Pointer) bool { return true }
func renvo_runtime_ChanSelect(cases []renvo_runtime_ChanSelectValue, hasDefault bool) int { return -1 }
func renvo_runtime_ChanClose(channel uintptr) {}
func renvo_runtime_ChanLen(channel uintptr) int { return 0 }
func renvo_runtime_ChanCap(channel uintptr) int { return 0 }
func renvo_runtime_Spawn(entry uintptr, context uintptr) {}
func renvo_runtime_Call(entry uintptr, context uintptr) { panic("invalid") }
func consume(value int) {}
func sendOnly(values chan<- int, value int) { values <- value }
func receiveOnly(values <-chan int) int { return <-values }
func main() {
`)
		for _, value := range data {
			switch value % 8 {
			case 0:
				source.WriteString(`{ values := make(chan int, 2); values <- 7; got := <-values; consume(got) }
`)
			case 1:
				source.WriteString(`{ values := make(chan int, 1); select { case got := <-values: consume(got); default: consume(0) } }
`)
			case 2:
				source.WriteString(`{ values := make(chan int, 1); select { case values <- 9: consume(1); default: consume(0) } }
`)
			case 3:
				source.WriteString(`{ captured := 3; go func(input int) { consume(input + captured) }(captured) }
`)
			case 4:
				source.WriteString("go consume(4)\n")
			case 5:
				source.WriteString(`{ values := make(chan int, 1); values <- 5; close(values); for got := range values { consume(got) } }
`)
			case 6:
				source.WriteString(`{ values := make(chan int, 1); sendOnly(values, 6); consume(receiveOnly(values)) }
`)
			case 7:
				source.WriteString(`{ first := make(chan int, 1); second := make(chan int, 1); select { case first <- 1: consume(1); case got, open := <-second: if open { consume(got) }; default: consume(0) } }
`)
			}
		}
		source.WriteString("}\n")
		result := buildFromFiles(t, []load.SourceFile{
			{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
			{Path: "/repo/case/cmd/app/main.go", Src: source.Bytes()},
		})
		linked := LinkBuildCore(result)
		if !linked.Ok {
			t.Fatalf("link failed: err=%d pkg=%d\n%s", linked.Error, linked.ErrorPackage, source.String())
		}
		for i := 0; i < len(linked.Program.Tokens); i++ {
			text := functionValueTokenText(&linked.Program, i)
			if text == "go" || text == "chan" || text == "select" || text == "<-" || text == "range" {
				t.Fatalf("linked program retained concurrency token %q\n%s", text, linked.Program.Text)
			}
		}
		if len(linked.Program.ConcurrencySites) != 0 {
			t.Fatalf("linked program retained %d concurrency sites", len(linked.Program.ConcurrencySites))
		}
	})
}
