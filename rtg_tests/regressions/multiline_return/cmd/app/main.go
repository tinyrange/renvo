package main

func values() (int, int, int, int) {
	return 1,
		2,
		3,
		4
}

func main() {
	a, b, c, d := values()
	if a+b+c+d != 10 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
