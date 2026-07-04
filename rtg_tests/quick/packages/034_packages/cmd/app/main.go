package main

import "example.com/rtgtests/quick/packages/case034/pkg/lib"

func main() {
	if lib.Score(28) == 47 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
