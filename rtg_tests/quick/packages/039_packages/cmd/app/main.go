package main

import "example.com/rtgtests/quick/packages/case039/pkg/lib"

func main() {
	if lib.Score(24) == 48 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
