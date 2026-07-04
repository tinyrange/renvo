package main

import "example.com/rtgtests/extended/packageinit/case073/pkg/lib"

func main() {
	if lib.Value() == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
