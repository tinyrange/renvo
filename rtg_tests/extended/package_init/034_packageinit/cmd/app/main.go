package main

import "example.com/rtgtests/extended/packageinit/case034/pkg/lib"

func main() {
	if lib.Value() == 11 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
