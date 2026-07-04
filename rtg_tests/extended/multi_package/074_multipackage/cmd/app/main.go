package main

import "example.com/rtgtests/extended/multipackage/case074/pkg/a"
import "example.com/rtgtests/extended/multipackage/case074/pkg/b"

func main() {
	if a.Value()+b.Value() == 25 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
