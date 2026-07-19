package main

import "example.com/renvotests/quick/packages/case025/pkg/lib"

func main() {
	if lib.Score(12) == 22 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
