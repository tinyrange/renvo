package main

import "example.com/renvotests/extended/packageinit/case094/pkg/lib"

func main() {
	if lib.Value() == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
