package main

import "example.com/renvotests/extended/packageinit/case132/pkg/lib"

func main() {
	if lib.Value() == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
