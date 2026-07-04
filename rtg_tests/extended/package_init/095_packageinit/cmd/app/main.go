package main

import "example.com/rtgtests/extended/packageinit/case095/pkg/lib"

func main() {
	if lib.Value() == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
