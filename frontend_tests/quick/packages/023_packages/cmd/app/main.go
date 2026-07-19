package main

import "example.com/renvotests/quick/packages/case023/pkg/lib"

func main() {
	if lib.Score(31) == 39 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
