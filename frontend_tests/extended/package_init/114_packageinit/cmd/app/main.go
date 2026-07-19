package main

import "example.com/renvotests/extended/packageinit/case114/pkg/lib"

func main() {
	if lib.Value() == 29 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
