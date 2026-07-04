package main

import "example.com/rtgtests/extended/packageinit/case090/pkg/lib"

func main() {
	if lib.Value() == 36 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
