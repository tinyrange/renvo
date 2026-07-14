package main

func calc(a int, b int, c int) int {
	total := (a + b) * c
	total = total - b
	total = total + a%5
	return total
}

func main() {
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if calc(6, 7, 11) == 137 {
			print("PASS\n")
			return
		}
	}

	print("FAIL\n")
}
