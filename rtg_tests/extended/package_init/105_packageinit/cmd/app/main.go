package main

import "example.com/rtgtests/extended/packageinit/case105/pkg/lib"

func main() {
	if lib.Value() == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
