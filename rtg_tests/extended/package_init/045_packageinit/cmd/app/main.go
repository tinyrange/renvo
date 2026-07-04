package main

import "example.com/rtgtests/extended/packageinit/case045/pkg/lib"

func main() {
	if lib.Value() == 22 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
