package main

import "example.com/rtgtests/extended/multipackage/case050/pkg/a"
import "example.com/rtgtests/extended/multipackage/case050/pkg/b"

func main() {
	if a.Value()+b.Value() == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
