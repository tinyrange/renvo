package main

import "example.com/rtgtests/regressions/issue150/native"

func main() {
	if native.Lookup() {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
