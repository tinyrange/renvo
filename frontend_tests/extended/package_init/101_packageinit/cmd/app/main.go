package main

import "example.com/renvotests/extended/packageinit/case101/pkg/lib"

func main() {
	if lib.Value() == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
