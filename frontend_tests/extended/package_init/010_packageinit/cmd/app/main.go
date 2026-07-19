package main

import "example.com/renvotests/extended/packageinit/case010/pkg/lib"

func main() {
	if lib.Value() == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
