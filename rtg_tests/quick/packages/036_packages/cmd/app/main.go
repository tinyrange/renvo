package main

import "example.com/rtgtests/quick/packages/case036/pkg/lib"

func main() {
	if lib.Score(9) == 30 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
