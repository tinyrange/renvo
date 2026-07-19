package main

func main() {
	values := [3]int{4, 8, 3}
	total := values[0] + values[1]*2 + values[2]*3
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if total == 29 {
			print("PASS\n")
			return
		}
	}

	print("FAIL\n")
}
