package main

import "example.com/rtgtests/quick/packages/case003/pkg/lib"

func main() {
	if lib.Score(18) == 29 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
