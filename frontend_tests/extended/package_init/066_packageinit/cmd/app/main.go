package main

import "example.com/renvotests/extended/packageinit/case066/pkg/lib"

func main() {
	if lib.Value() == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
