package main

import "example.com/rtgtests/extended/packageinit/case017/pkg/lib"

func main() {
	if lib.Value() == 25 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
