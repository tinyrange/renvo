package main

import "example.com/renvotests/extended/packageinit/case002/pkg/lib"

func main() {
	if lib.Value() == 10 {
		print("PASS\n")
		return
	} else {

		print("FAIL\n")
	}
}
