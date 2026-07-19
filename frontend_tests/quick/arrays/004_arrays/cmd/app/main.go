package main

func main() {
	values := [3]int{5, 10, 6}
	total := values[0] + values[1]*2 + values[2]*3
	corpusOK := false
	if total == 43 {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
