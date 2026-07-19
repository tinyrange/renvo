package main

import "example.com/renvotests/quick/packages/case030/pkg/lib"

func main() {
	if lib.Score(8) == 23 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
