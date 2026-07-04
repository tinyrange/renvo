package main

import "example.com/rtgtests/extended/multipackage/case119/pkg/a"
import "example.com/rtgtests/extended/multipackage/case119/pkg/b"

func main() {
	if a.Value()+b.Value() == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
