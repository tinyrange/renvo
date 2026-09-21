package main

func f() (x int) {
	if true { x := 1; return }; return
}

func main() {}
