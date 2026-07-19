package main

import "example.com/renvotests/extended/packageinit/case060/pkg/lib"

func main() {
	if lib.Value() == 37 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
