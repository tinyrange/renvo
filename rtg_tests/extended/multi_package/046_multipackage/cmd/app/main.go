package main

import "example.com/rtgtests/extended/multipackage/case046/pkg/a"
import "example.com/rtgtests/extended/multipackage/case046/pkg/b"

func main() {
	if a.Value()+b.Value() == 11 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
