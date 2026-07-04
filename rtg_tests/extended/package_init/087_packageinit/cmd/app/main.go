package main

import "example.com/rtgtests/extended/packageinit/case087/pkg/lib"

func main() {
	if lib.Value() == 33 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
