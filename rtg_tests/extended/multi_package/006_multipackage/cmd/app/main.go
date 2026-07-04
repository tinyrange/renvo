package main

import "example.com/rtgtests/extended/multipackage/case006/pkg/a"
import "example.com/rtgtests/extended/multipackage/case006/pkg/b"

func main() {
	if a.Value()+b.Value() == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
