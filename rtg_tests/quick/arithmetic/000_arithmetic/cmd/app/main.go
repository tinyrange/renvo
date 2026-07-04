package main

func calc(a int, b int, c int) int {
	total := (a + b) * c
	total = total - b
	total = total + a%5
	return total
}

func main() {
	if calc(3, 5, 2) == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
