package main

import "example.com/renvotests/extended/packageinit/case019/pkg/lib"

func main() {
	if lib.Value() == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
