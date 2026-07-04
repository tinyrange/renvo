package main

import "example.com/rtgtests/quick/packages/case018/pkg/lib"

func main() {
	if lib.Score(6) == 32 {
		print(lib.Text())
		return
	}
	print("FAIL\n")
}
