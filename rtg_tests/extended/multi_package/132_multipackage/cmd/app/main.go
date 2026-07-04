package main

import "example.com/rtgtests/extended/multipackage/case132/pkg/a"
import "example.com/rtgtests/extended/multipackage/case132/pkg/b"

func main() {
	if a.Value()+b.Value() == 38 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
