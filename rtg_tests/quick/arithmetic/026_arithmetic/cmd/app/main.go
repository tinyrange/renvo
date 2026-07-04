package main

func calc(a int, b int, c int) int {
	total := (a + b) * c
	total = total - b
	total = total + a%5
	return total
}

func main() {
	if calc(12, 16, 3) == 70 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
