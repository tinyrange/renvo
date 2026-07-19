package main

import "example.com/renvotests/extended/packageinit/case030/pkg/lib"

func main() {
	if lib.Value() == 38 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
