package main

import "example.com/rtgtests/quick/packages/case037/pkg/lib"

func main() {
	if lib.Score(14) == 36 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
