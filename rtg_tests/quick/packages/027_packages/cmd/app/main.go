package main

import "example.com/rtgtests/quick/packages/case027/pkg/lib"

func main() {
	if lib.Score(22) == 34 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
