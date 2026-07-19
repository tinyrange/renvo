package main

import "example.com/renvotests/extended/packageinit/case116/pkg/lib"

func main() {
	if lib.Value() == 31 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
