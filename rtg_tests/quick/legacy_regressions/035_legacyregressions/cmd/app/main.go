package main

func main() {
	xs := []int{1, 2}
	ys := xs
	if len(ys) == 2 && cap(ys) == 2 && ys[0] == 1 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
