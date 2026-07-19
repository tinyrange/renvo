package main

import "example.com/renvotests/extended/packageinit/case084/pkg/lib"

func main() {
	if lib.Value() == 30 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
