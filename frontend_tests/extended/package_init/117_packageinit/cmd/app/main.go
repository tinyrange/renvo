package main

import "example.com/renvotests/extended/packageinit/case117/pkg/lib"

func main() {
	if lib.Value() == 32 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
