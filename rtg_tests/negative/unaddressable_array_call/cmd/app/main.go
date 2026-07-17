package main

func produce() [2]int { return [2]int{1, 2} }

func main() {
	part := produce()[:]
	_ = part
}
