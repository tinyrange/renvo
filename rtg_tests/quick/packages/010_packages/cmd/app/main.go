package main

import "example.com/rtgtests/quick/packages/case010/pkg/lib"

func main() {
	if lib.Score(24) == 42 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
