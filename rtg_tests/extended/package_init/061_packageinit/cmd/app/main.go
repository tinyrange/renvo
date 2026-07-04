package main

import "example.com/rtgtests/extended/packageinit/case061/pkg/lib"

func main() {
	if lib.Value() == 38 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
