package main

import "example.com/renvotests/extended/packageinit/case001/pkg/lib"

func main() {
	corpusOK := lib.Value() == 9
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
