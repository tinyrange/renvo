package main

import "example.com/renvotests/quick/packages/case008/pkg/lib"

func main() {
	if lib.Score(14) == 30 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
