package main

import "example.com/renvotests/quick/packages/case024/pkg/lib"

func main() {
	if lib.Score(7) == 16 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
