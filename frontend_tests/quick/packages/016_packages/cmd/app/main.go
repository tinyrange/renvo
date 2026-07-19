package main

import "example.com/renvotests/quick/packages/case016/pkg/lib"

func main() {
	if lib.Score(25) == 49 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
