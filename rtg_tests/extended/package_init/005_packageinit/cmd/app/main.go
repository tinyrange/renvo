package main

import "example.com/rtgtests/extended/packageinit/case005/pkg/lib"

func main() {
	if lib.Value() == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
