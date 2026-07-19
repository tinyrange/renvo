package main

import "example.com/renvotests/extended/packageinit/case033/pkg/lib"

func main() {
	if lib.Value() == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
