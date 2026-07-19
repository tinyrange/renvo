package main

func main() {
	n := 0
	for range []int{1, 2} {
		n++
	}
	if n == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
