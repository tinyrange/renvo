package main

import "example.com/rtgtests/extended/multipackage/case136/pkg/a"
import "example.com/rtgtests/extended/multipackage/case136/pkg/b"

func main() {
	if a.Value()+b.Value() == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
