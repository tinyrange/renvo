package main

import "example.com/rtgtests/extended/multipackage/case036/pkg/a"
import "example.com/rtgtests/extended/multipackage/case036/pkg/b"

func main() {
	if a.Value()+b.Value() == 33 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
