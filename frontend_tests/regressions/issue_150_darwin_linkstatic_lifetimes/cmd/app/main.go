package main

import "example.com/renvotests/regressions/issue150/native"

func main() {
	if native.Lookup() {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
