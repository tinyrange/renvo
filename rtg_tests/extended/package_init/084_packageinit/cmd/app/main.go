package main

import "example.com/rtgtests/extended/packageinit/case084/pkg/lib"

func main() {
	if lib.Value() == 30 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
