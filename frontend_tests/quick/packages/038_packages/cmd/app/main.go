package main

import "example.com/renvotests/quick/packages/case038/pkg/lib"

func main() {
	if lib.Score(19) == 42 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
