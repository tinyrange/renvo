package main

func main() {
	total := 0
	for i := 0; i < 9; i++ {
		if i%5 == 0 {
			continue
		}
		if i > 9-2 {
			break
		}
		total = total + i
	}
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if total == 23 {
			print("PASS\n")
			return
		}
	}

	print("FAIL\n")
}
