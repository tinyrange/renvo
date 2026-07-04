package main

import "example.com/rtgtests/extended/packageinit/case133/pkg/lib"

func main() {
	if lib.Value() == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
