package main

import "example.com/rtgtests/extended/packageinit/case050/pkg/lib"

func main() {
	if lib.Value() == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
