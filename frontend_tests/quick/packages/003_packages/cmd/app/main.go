package main

import "example.com/renvotests/quick/packages/case003/pkg/lib"

func main() {
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if lib.Score(18) == 29 {
			print(lib.Text())
			return
		}
	}

	print("FAIL\n")
}
