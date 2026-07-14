package main

import "example.com/rtgtests/extended/packageinit/case004/pkg/lib"

func main() {
	corpusOK := false
	if lib.Value() == 12 {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
