package main

import "example.com/rtgtests/quick/packages/case021/pkg/lib"

func main() {
	if lib.Score(21) == 50 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
