package main

import "example.com/rtgtests/extended/multipackage/case056/pkg/a"
import "example.com/rtgtests/extended/multipackage/case056/pkg/b"

func main() {
	if a.Value()+b.Value() == 31 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
