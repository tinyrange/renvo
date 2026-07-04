package main

import "example.com/rtgtests/extended/multipackage/case052/pkg/a"
import "example.com/rtgtests/extended/multipackage/case052/pkg/b"

func main() {
	if a.Value()+b.Value() == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
