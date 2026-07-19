package main

import "example.com/renvotests/quick/packages/case032/pkg/lib"

func main() {
	if lib.Score(18) == 35 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
