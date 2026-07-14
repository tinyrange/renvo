package main

import "example.com/rtgtests/quick/packages/case002/pkg/lib"

func main() {
	if lib.Score(13) == 23 {
		print(lib.Text())
		return
	} else {

		print("FAIL\n")
	}
}
