package main

import "example.com/renvotests/extended/packageinit/case149/pkg/lib"

func main() {
	if lib.Value() == 33 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
