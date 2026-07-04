package main

import "example.com/rtgtests/extended/multipackage/case019/pkg/a"
import "example.com/rtgtests/extended/multipackage/case019/pkg/b"

func main() {
	if a.Value()+b.Value() == 22 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
