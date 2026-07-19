package main

import "example.com/renvotests/quick/packages/case026/pkg/lib"

func main() {
	if lib.Score(17) == 28 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
