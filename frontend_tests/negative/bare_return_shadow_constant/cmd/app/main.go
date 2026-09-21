package main

func f() (x int) {
	if true { const x = 1; return }; return
}

func main() {}
