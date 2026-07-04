package main

import "example.com/rtgtests/quick/packages/case020/pkg/lib"

func main() {
	if lib.Score(16) == 44 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
