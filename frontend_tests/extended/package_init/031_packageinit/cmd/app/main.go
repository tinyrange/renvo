package main

import "example.com/renvotests/extended/packageinit/case031/pkg/lib"

func main() {
	if lib.Value() == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
