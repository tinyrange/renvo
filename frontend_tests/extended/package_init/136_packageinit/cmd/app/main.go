package main

import "example.com/renvotests/extended/packageinit/case136/pkg/lib"

func main() {
	if lib.Value() == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
