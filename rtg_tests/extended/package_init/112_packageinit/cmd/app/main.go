package main

import "example.com/rtgtests/extended/packageinit/case112/pkg/lib"

func main() {
	if lib.Value() == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
