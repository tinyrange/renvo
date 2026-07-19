package main

import "example.com/renvotests/extended/packageinit/case110/pkg/lib"

func main() {
	if lib.Value() == 25 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
