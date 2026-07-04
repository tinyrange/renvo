package main

import "example.com/rtgtests/extended/packageinit/case120/pkg/lib"

func main() {
	if lib.Value() == 35 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
