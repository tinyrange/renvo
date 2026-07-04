package main

import "example.com/rtgtests/quick/packages/case031/pkg/lib"

func main() {
	if lib.Score(13) == 29 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
