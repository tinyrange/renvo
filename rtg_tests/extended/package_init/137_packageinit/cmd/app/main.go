package main

import "example.com/rtgtests/extended/packageinit/case137/pkg/lib"

func main() {
	if lib.Value() == 21 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
