package main

import "example.com/rtgtests/extended/multipackage/case057/pkg/a"
import "example.com/rtgtests/extended/multipackage/case057/pkg/b"

func main() {
	if a.Value()+b.Value() == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
