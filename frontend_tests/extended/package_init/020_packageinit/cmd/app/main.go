package main

import "example.com/renvotests/extended/packageinit/case020/pkg/lib"

func main() {
	if lib.Value() == 28 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
