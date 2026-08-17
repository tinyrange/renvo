package main

import (
	"example.com/renvotests/regressions/goroutine_channel_runtime/pipes"
	runtime "renvo.dev/x/runtime"
	"renvo.dev/x/runtime/serial"
)

func produce(values chan<- int) {
	values <- 42
	close(values)
}

func sendResult(values chan<- int, value int) { values <- value }

type resultCall func(chan<- int, int)

type callHolder struct{ call resultCall }

func launchFunctionValue(call resultCall, values chan<- int, value int) {
	go call(values, value)
}

func closeOnReturn(values chan int) { defer close(values) }

func expectClosedSend(values chan int, result chan<- int) {
	defer func() {
		if recover() == nil {
			result <- -1
		} else {
			result <- 1
		}
	}()
	values <- 1
}

func expectClosedClose(values chan int, result chan<- int) {
	defer func() {
		if recover() == nil {
			result <- -1
		} else {
			result <- 1
		}
	}()
	close(values)
}

func recoverValue(values chan<- int, wanted int) {
	defer func() {
		value, ok := recover().(int)
		if !ok || value != wanted {
			sendResult(values, -1)
			return
		}
		sendResult(values, value)
	}()
	panic(wanted)
}

type packet struct {
	value int
	text  string
	pair  [2]int
}

type worker struct{ base int }

type numbered interface{ Number() int }

func (p packet) Number() int { return p.value }

func (w worker) send(values chan<- int, value int) { values <- w.base + value }

func launchAfterReturn(values chan<- int) {
	value := 31
	go func() { values <- value }()
}

func launchShared(values chan<- int, gate <-chan int) {
	value := 1
	go func() {
		<-gate
		values <- value
	}()
	value = 2
}

func choose(values chan<- int, first <-chan int, second <-chan int) {
	select {
	case value := <-first:
		values <- value
	case value := <-second:
		values <- value
	}
}

var evaluation int
var goEvaluation int

func evaluatedChannel(values chan int) chan int {
	evaluation = evaluation*10 + 1
	return values
}

func evaluatedValue() int {
	evaluation = evaluation*10 + 2
	return 12
}

func evaluatedGoChannel(values chan<- int) chan<- int {
	goEvaluation = goEvaluation*10 + 1
	return values
}

func evaluatedGoValue() int {
	goEvaluation = goEvaluation*10 + 2
	return 13
}

func main() {
	handler := serial.New()
	runtime.EnableGoroutines(handler)

	values := make(chan int)
	go produce(values)
	total := 0
	for value := range values {
		total += value
	}
	if total != 42 {
		print("FAIL range\n")
		return
	}
	other := make(chan int)
	go func() { other <- 7 }()
	select {
	case value := <-other:
		if value != 7 {
			print("FAIL select value\n")
			return
		}
	default:
		// Native Go may not have scheduled the producer yet. The Renvo serial
		// handler blocks here instead, so retry without a default.
		if value := <-other; value != 7 {
			print("FAIL fallback value\n")
			return
		}
	}

	// A select without a default must suspend on its own stack. Completing one
	// case removes the sibling registration before either sender can continue.
	firstChoice := make(chan int)
	secondChoice := make(chan int)
	chosen := make(chan int)
	go choose(chosen, firstChoice, secondChoice)
	go func() { firstChoice <- 19 }()
	go func() { secondChoice <- 23 }()
	selected := <-chosen
	if selected != 19 && selected != 23 {
		print("FAIL blocking select\n")
		return
	}
	if selected == 19 {
		if value := <-secondChoice; value != 23 {
			print("FAIL select sibling\n")
			return
		}
	} else if value := <-firstChoice; value != 19 {
		print("FAIL select sibling\n")
		return
	}

	buffered := make(chan int, 2)
	buffered <- 3
	if len(buffered) != 1 || cap(buffered) != 2 {
		print("FAIL buffer size\n")
		return
	}
	value, open := <-buffered
	close(buffered)
	_, closedOpen := <-buffered
	if value != 3 || !open || closedOpen {
		print("FAIL close\n")
		return
	}

	// Select operands are evaluated exactly once and in source order, even
	// when the default or another case could be selected.
	ordered := make(chan int, 1)
	select {
	case evaluatedChannel(ordered) <- evaluatedValue():
	default:
		print("FAIL ready select\n")
		return
	}
	if evaluation != 12 {
		print("FAIL evaluation order\n")
		return
	}
	var nilValues chan int
	select {
	case nilValues <- 1:
		print("FAIL nil select\n")
		return
	case <-nilValues:
		print("FAIL nil select\n")
		return
	default:
	}
	orderedValue := <-ordered
	if orderedValue != 12 {
		print("FAIL evaluation order\n")
		return
	}

	packets := make(chan packet)
	go func() { packets <- packet{value: 5, text: "typed", pair: [2]int{7, 11}} }()
	packetValue, packetOpen := <-packets
	if !packetOpen {
		print("FAIL typed copy\n")
		return
	}
	if packetValue.value != 5 {
		print("FAIL typed copy\n")
		return
	}
	if packetValue.text != "typed" {
		print("FAIL typed copy\n")
		return
	}
	firstPacketValue := packetValue.pair[0]
	secondPacketValue := packetValue.pair[1]
	if firstPacketValue+secondPacketValue != 18 {
		print("FAIL typed copy\n")
		return
	}

	interfaces := make(chan numbered, 1)
	interfaces <- packet{value: 37}
	interfaceValue := <-interfaces
	if interfaceValue.Number() != 37 {
		print("FAIL interface copy\n")
		return
	}
	slices := make(chan []int, 1)
	slices <- []int{2, 3, 5}
	sliceValue := <-slices
	if len(sliceValue) != 3 || sliceValue[0]+sliceValue[1]+sliceValue[2] != 10 {
		print("FAIL slice copy\n")
		return
	}

	launched := make(chan int)
	launchAfterReturn(launched)
	launchedValue := <-launched
	if launchedValue != 31 {
		print("FAIL persistent context\n")
		return
	}

	shared := make(chan int)
	gate := make(chan int)
	launchShared(shared, gate)
	gate <- 1
	if sharedValue := <-shared; sharedValue != 2 {
		print("FAIL shared capture\n")
		return
	}

	dynamic := make(chan int, 3)
	call := sendResult
	go call(dynamic, 2)
	item := worker{base: 10}
	go item.send(dynamic, 3)
	go func(value int) { dynamic <- value }(17)
	dynamicTotal := <-dynamic + <-dynamic + <-dynamic
	if dynamicTotal != 32 {
		print("FAIL dynamic calls\n")
		return
	}
	callValue := callHolder{}
	callValue.call = sendResult
	methodValue := callHolder{}
	methodValue.call = item.send
	launchFunctionValue(callValue.call, dynamic, 4)
	launchFunctionValue(methodValue.call, dynamic, 5)
	go sendResult(evaluatedGoChannel(dynamic), evaluatedGoValue())
	stagedFirst := <-dynamic
	stagedSecond := <-dynamic
	stagedThird := <-dynamic
	if stagedFirst+stagedSecond+stagedThird != 32 || goEvaluation != 12 {
		print("FAIL staged go calls\n")
		return
	}

	crossPackage := pipes.New()
	pipes.Send(crossPackage, 29)
	if pipes.Receive(crossPackage) != 29 {
		print("FAIL named directional channel\n")
		return
	}
	deferred := make(chan int)
	go closeOnReturn(deferred)
	if _, open := <-deferred; open {
		print("FAIL deferred close\n")
		return
	}
	closed := make(chan int)
	close(closed)
	panicResults := make(chan int, 2)
	go expectClosedSend(closed, panicResults)
	go expectClosedClose(closed, panicResults)
	panicFirst := <-panicResults
	panicSecond := <-panicResults
	if panicFirst+panicSecond != 2 {
		print("FAIL channel panic\n")
		return
	}

	panics := make(chan int, 2)
	go recoverValue(panics, 11)
	go recoverValue(panics, 22)
	first := <-panics
	second := <-panics
	if first+second != 33 || first == second {
		print("FAIL thread state\n")
		return
	}

	stress := make(chan int, 16)
	for i := 0; i < 16; i++ {
		go sendResult(stress, i)
	}
	stressTotal := 0
	for i := 0; i < 16; i++ {
		stressTotal += <-stress
	}
	if stressTotal != 120 {
		print("FAIL stress\n")
		return
	}
	print("PASS\n")
}
