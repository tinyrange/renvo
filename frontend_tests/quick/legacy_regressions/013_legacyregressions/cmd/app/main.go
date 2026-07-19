package main

func issue13Mutate(a [2]int) int { a[0] = 9; return a[0] + a[1] }

func main() {
	a := [2]int{1, 2}
	if issue13Mutate(a) == 11 && a[0] == 1 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
