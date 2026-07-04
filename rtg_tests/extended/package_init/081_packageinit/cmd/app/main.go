package main

import "example.com/rtgtests/extended/packageinit/case081/pkg/lib"

func main() {
	if lib.Value() == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
