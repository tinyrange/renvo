package main

import "example.com/rtgtests/extended/packageinit/case052/pkg/lib"

func main() {
	if lib.Value() == 29 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
