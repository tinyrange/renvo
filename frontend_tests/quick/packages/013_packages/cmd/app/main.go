package main

import "example.com/renvotests/quick/packages/case013/pkg/lib"

func main() {
	if lib.Score(10) == 31 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
