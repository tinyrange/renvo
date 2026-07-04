package main

import "example.com/rtgtests/extended/packageinit/case018/pkg/lib"

func main() {
	if lib.Value() == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
