package main

func main() {
	xs := make([]int, 1, 2)
	xs = append(xs, 8)
	xs = append(xs, 9)
	if len(xs) == 3 && cap(xs) >= 3 && xs[2] == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
