package main

import "example.com/rtgtests/extended/multipackage/case135/pkg/a"
import "example.com/rtgtests/extended/multipackage/case135/pkg/b"

func main() {
	if a.Value()+b.Value() == 25 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
