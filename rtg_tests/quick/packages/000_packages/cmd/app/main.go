package main

import "example.com/rtgtests/quick/packages/case000/pkg/lib"

func main() {
	if lib.Score(3) == 11 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
