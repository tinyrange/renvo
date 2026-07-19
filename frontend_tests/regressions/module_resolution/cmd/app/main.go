package main

import "example.com/renvotests/regressions/module_resolution/lib"

func main() {
	if lib.Value() == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
