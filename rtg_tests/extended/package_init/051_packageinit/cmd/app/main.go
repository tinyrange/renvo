package main

import "example.com/rtgtests/extended/packageinit/case051/pkg/lib"

func main() {
	if lib.Value() == 28 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
