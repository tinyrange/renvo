package main

import "example.com/rtgtests/extended/packageinit/case077/pkg/lib"

func main() {
	if lib.Value() == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
