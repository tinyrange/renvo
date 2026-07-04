package main

import "example.com/rtgtests/quick/packages/case001/pkg/lib"

func main() {
	if lib.Score(8) == 17 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
