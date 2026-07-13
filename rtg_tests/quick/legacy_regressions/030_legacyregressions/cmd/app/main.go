package main

func main() {
	xs := []int{1, 2}
	xs[0], xs[1] = xs[1], xs[0]
	if xs[0] == 2 && xs[1] == 1 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
