package main

func calc(a int, b int, c int) int {
	total := (a + b) * c
	total = total - b
	total = total + a%5
	return total
}

func main() {
	corpusOK := calc(4, 12, 5) == 72
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
