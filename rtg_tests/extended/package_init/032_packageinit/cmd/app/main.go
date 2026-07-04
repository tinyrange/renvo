package main

import "example.com/rtgtests/extended/packageinit/case032/pkg/lib"

func main() {
	if lib.Value() == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
