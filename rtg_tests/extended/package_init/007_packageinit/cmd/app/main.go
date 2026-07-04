package main

import "example.com/rtgtests/extended/packageinit/case007/pkg/lib"

func main() {
	if lib.Value() == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
