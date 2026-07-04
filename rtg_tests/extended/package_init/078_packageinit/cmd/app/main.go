package main

import "example.com/rtgtests/extended/packageinit/case078/pkg/lib"

func main() {
	if lib.Value() == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
