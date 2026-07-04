package main

import "example.com/rtgtests/extended/packageinit/case091/pkg/lib"

func main() {
	if lib.Value() == 37 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
