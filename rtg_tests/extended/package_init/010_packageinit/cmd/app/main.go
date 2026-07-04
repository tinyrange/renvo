package main

import "example.com/rtgtests/extended/packageinit/case010/pkg/lib"

func main() {
	if lib.Value() == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
