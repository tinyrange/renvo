package main

func main() {
	values := [3]int{2, 4, 6}
	total := values[0] + values[1]*2 + values[2]*3
	corpusOK := total == 28
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
