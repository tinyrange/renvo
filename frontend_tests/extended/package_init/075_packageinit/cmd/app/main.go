package main

import "example.com/renvotests/extended/packageinit/case075/pkg/lib"

func main() {
	if lib.Value() == 21 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
