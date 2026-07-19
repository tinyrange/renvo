package main

import "example.com/renvotests/quick/packages/case004/pkg/lib"

func main() {
	corpusOK := false
	if lib.Score(23) == 35 {
		corpusOK = true
	}
	if corpusOK {
		print(lib.Text())
		return
	}

	print("FAIL\n")
}
