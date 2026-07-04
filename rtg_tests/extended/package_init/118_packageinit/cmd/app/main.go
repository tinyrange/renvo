package main

import "example.com/rtgtests/extended/packageinit/case118/pkg/lib"

func main() {
	if lib.Value() == 33 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
