package main

import "example.com/rtgtests/quick/packages/case004/pkg/lib"

func main() {
	if lib.Score(23) == 35 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
