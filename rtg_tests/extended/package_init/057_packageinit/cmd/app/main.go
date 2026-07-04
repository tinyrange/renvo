package main

import "example.com/rtgtests/extended/packageinit/case057/pkg/lib"

func main() {
	if lib.Value() == 34 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
