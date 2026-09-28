package main

var initialized = make(chan int, 2)

func init() { initialized <- 3; initialized <- 4; close(initialized) }

func main() {
	total := 0
	for value := range initialized {
		total += value
	}
	if total != 7 {
		panic("package initialization")
	}
	if value, open := <-initialized; value != 0 || open {
		panic("closed receive")
	}
	done := make(chan int)
	go func() { done <- 42 }()
	if <-done != 42 {
		panic("rendezvous")
	}
	c := make(chan int, 1)
	select {
	case <-c:
		panic("empty channel")
	default:
	}
	c <- 7
	select {
	case value := <-c:
		if value != 7 {
			panic("select value")
		}
	default:
		panic("ready channel")
	}
	if len(c) != 0 || cap(c) != 1 {
		panic("channel size")
	}
	println("PASS")
}
