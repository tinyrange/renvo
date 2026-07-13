package main

func main() {
	a := [2]int{1, 2}
	a[0], a[1] = 9, 8
	if a[0] == 9 && a[1] == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
