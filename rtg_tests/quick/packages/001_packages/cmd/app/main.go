package main

import "example.com/rtgtests/quick/packages/case001/pkg/lib"

func main() {
	corpusOK := lib.Score(8) == 17
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print(lib.Text())

}
