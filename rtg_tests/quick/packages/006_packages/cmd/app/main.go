package main

import "example.com/rtgtests/quick/packages/case006/pkg/lib"

func main() {
	if lib.Score(4) == 18 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
