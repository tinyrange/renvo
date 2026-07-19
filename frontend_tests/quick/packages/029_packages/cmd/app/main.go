package main

import "example.com/renvotests/quick/packages/case029/pkg/lib"

func main() {
	if lib.Score(3) == 17 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
