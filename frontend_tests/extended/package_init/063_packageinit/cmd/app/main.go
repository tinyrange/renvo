package main

import "example.com/renvotests/extended/packageinit/case063/pkg/lib"

func main() {
	if lib.Value() == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
