package main

func main() {
	values := []int{4, 5, 6}
	values = append(values[1:2], 7)
	corpusOK := false
	if len(values) == 2 && values[0]+values[1] == 12 {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
