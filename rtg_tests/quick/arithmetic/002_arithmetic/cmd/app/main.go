package main

func calc(a int, b int, c int) int {
	total := (a + b) * c
	total = total - b
	total = total + a%5
	return total
}

func main() {
	if calc(5, 19, 8) == 173 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
