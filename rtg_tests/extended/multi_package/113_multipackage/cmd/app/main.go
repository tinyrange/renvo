package main

import "example.com/rtgtests/extended/multipackage/case113/pkg/a"
import "example.com/rtgtests/extended/multipackage/case113/pkg/b"

func main() {
	if a.Value()+b.Value() == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
