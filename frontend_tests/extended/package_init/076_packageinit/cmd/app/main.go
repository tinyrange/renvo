package main

import "example.com/renvotests/extended/packageinit/case076/pkg/lib"

func main() {
	if lib.Value() == 22 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
