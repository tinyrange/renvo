package main

func main() {
	xs := []int{1, 2}
	ys := append(xs, 3)
	if len(ys) == 3 && ys[2] == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
