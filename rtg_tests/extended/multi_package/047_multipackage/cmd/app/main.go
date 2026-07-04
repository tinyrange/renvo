package main

import "example.com/rtgtests/extended/multipackage/case047/pkg/a"
import "example.com/rtgtests/extended/multipackage/case047/pkg/b"

func main() {
	if a.Value()+b.Value() == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
