package main

import "example.com/rtgtests/quick/packages/case028/pkg/lib"

func main() {
	if lib.Score(27) == 40 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
