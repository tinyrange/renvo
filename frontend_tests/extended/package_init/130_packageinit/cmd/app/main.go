package main

import "example.com/renvotests/extended/packageinit/case130/pkg/lib"

func main() {
	if lib.Value() == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
