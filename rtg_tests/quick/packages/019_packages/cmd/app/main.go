package main

import "example.com/rtgtests/quick/packages/case019/pkg/lib"

func main() {
	if lib.Score(11) == 38 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
