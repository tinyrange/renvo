package main

import "example.com/renvotests/extended/packageinit/case006/pkg/lib"

func main() {
	if lib.Value() == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
