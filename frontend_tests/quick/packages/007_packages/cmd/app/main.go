package main

import "example.com/renvotests/quick/packages/case007/pkg/lib"

func main() {
	if lib.Score(9) == 24 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
