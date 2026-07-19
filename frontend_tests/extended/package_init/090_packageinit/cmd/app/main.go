package main

import "example.com/renvotests/extended/packageinit/case090/pkg/lib"

func main() {
	if lib.Value() == 36 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
