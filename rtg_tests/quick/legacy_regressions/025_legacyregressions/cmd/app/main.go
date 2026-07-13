package main

type issue25S struct{ n int }

func main() {
	var a, b = issue25S{n: 1}, issue25S{n: 2}
	if a.n == 1 && b.n == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
