package main

import "example.com/renvotests/extended/packageinit/case141/pkg/lib"

func main() {
	if lib.Value() == 25 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
