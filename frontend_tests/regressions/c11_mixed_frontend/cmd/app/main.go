package main

func goDouble(value int) int { return value * 2 }

func main() {
	if cAdd(20, 22) == 42 && cCallGo(21) == 42 {
		print("PASS\n")
	}
}
