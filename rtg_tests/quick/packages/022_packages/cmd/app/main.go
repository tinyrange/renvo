package main

import "example.com/rtgtests/quick/packages/case022/pkg/lib"

func main() {
	if lib.Score(26) == 56 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
