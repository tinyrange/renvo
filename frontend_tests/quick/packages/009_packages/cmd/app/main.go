package main

import "example.com/renvotests/quick/packages/case009/pkg/lib"

func main() {
	if lib.Score(19) == 36 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
