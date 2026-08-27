package main

import (
	"example.com/renvotests/regressions/colliding_imported_method_internal_call/first"
	"example.com/renvotests/regressions/colliding_imported_method_internal_call/second"
)

func main() {
	firstDevice := first.Device{}
	value, err := firstDevice.Measure()
	secondDevice := second.Device{}
	if value != 42 || err != nil || secondDevice.Read() != 7 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
