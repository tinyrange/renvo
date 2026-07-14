package main

import "example.com/rtgtests/extended/multipackage/case003/pkg/a"
import "example.com/rtgtests/extended/multipackage/case003/pkg/b"

func main() {
	for corpusAttempt := 0; corpusAttempt < 1; corpusAttempt++ {
		if a.Value()+b.Value() == 9 {
			print("PASS\n")
			return
		}
	}

	print("FAIL\n")
}
