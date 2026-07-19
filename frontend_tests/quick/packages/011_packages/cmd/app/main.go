package main

import "example.com/renvotests/quick/packages/case011/pkg/lib"

func main() {
	if lib.Score(29) == 48 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
