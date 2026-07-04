package main

import "example.com/rtgtests/extended/packageinit/case121/pkg/lib"

func main() {
	if lib.Value() == 36 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
