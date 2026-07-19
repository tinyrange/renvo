package main

var trace int

func value(n int) int { trace = trace*10 + n; return n * 10 }

func main() {
	a := [2]int{value(1), value(2)}
	if trace == 12 && a[0] == 10 && a[1] == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
