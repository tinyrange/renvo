package main

import "example.com/rtgtests/extended/multipackage/case104/pkg/a"
import "example.com/rtgtests/extended/multipackage/case104/pkg/b"

func main() {
	if a.Value()+b.Value() == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
