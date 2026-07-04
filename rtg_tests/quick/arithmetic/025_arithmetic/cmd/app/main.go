package main

func calc(a int, b int, c int) int {
	total := (a + b) * c
	total = total - b
	total = total + a%5
	return total
}

func main() {
	if calc(11, 9, 11) == 212 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
