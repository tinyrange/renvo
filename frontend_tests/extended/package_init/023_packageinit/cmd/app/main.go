package main

import "example.com/renvotests/extended/packageinit/case023/pkg/lib"

func main() {
	if lib.Value() == 31 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
