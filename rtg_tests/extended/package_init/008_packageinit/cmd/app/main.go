package main

import "example.com/rtgtests/extended/packageinit/case008/pkg/lib"

func main() {
	if lib.Value() == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
