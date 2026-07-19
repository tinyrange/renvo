package main

import "example.com/renvotests/quick/packages/case033/pkg/lib"

func main() {
	if lib.Score(23) == 41 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
