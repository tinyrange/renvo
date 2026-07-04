package main

import "example.com/rtgtests/extended/packageinit/case035/pkg/lib"

func main() {
	if lib.Value() == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
