package main

import "example.com/renvotests/extended/packageinit/case050/pkg/lib"

func main() {
	if lib.Value() == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
