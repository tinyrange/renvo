package main

func read(ch <-chan int) int {
	return <-ch
}

func main() {
	ch := make(chan int, 2)
	ch <- 1
	select {
	case ch <- 2:
	default:
		panic("send")
	}
	if <-(ch) != 1 || read(ch) != 2 {
		panic("receive")
	}
	channels := []chan int{ch}
	ch <- 3
	if <-channels[0] != 3 {
		panic("indexed receive")
	}
	text := make(chan string, 1)
	text <- "value"
	select {
	case value := <-text:
		if value != "value" {
			panic("text")
		}
	default:
		panic("select receive")
	}
	println("PASS")
}
