package main

import "example.com/renvotests/extended/packageinit/case015/pkg/lib"

func main() {
	if lib.Value() == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
