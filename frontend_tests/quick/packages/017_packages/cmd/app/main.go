package main

import "example.com/renvotests/quick/packages/case017/pkg/lib"

func main() {
	if lib.Score(30) == 55 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
