package main

import "example.com/renvotests/extended/packageinit/case018/pkg/lib"

func main() {
	if lib.Value() == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
