package main

import "example.com/renvotests/quick/packages/case005/pkg/lib"

func main() {
	if lib.Score(28) == 41 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
