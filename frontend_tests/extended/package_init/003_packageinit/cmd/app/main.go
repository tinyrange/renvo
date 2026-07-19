package main

import "example.com/renvotests/extended/packageinit/case003/pkg/lib"

func main() {
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if lib.Value() == 11 {
			print("PASS\n")
			return
		}
	}

	print("FAIL\n")
}
