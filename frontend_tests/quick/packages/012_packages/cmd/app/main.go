package main

import "example.com/renvotests/quick/packages/case012/pkg/lib"

func main() {
	if lib.Score(5) == 25 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
