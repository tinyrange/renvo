package main

func calc(a int, b int, c int) int {
	total := (a + b) * c
	total = total - b
	total = total + a%5
	return total
}

func main() {
	if calc(6, 17, 3) == 53 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
