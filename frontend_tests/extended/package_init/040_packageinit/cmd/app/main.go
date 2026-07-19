package main

import "example.com/renvotests/extended/packageinit/case040/pkg/lib"

func main() {
	if lib.Value() == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
