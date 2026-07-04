package main

import "example.com/rtgtests/quick/packages/case015/pkg/lib"

func main() {
	if lib.Score(20) == 43 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
