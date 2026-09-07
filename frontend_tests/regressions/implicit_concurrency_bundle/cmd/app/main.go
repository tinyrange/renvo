package main

func main() {
	values := make(chan int, 1)
	values <- 7
	if <-values != 7 {
		panic("buffered channel")
	}
	go func() { values <- 11 }()
	if <-values != 11 {
		panic("goroutine handoff")
	}
	close(values)
	if _, ok := <-values; ok {
		panic("closed channel")
	}
	print("PASS\n")
}
