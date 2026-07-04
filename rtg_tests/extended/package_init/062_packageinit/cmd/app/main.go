package main

import "example.com/rtgtests/extended/packageinit/case062/pkg/lib"

func main() {
	if lib.Value() == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
