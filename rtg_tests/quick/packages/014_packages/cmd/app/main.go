package main

import "example.com/rtgtests/quick/packages/case014/pkg/lib"

func main() {
	if lib.Score(15) == 37 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
