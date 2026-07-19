package main

import "example.com/renvotests/quick/packages/case035/pkg/lib"

func main() {
	if lib.Score(4) == 24 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
