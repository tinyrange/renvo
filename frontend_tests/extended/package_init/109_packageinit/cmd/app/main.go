package main

import "example.com/renvotests/extended/packageinit/case109/pkg/lib"

func main() {
	if lib.Value() == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
