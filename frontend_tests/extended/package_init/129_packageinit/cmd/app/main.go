package main

import "example.com/renvotests/extended/packageinit/case129/pkg/lib"

func main() {
	if lib.Value() == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
