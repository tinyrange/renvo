package main

import "example.com/renvotests/extended/packageinit/case011/pkg/lib"

func main() {
	if lib.Value() == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
