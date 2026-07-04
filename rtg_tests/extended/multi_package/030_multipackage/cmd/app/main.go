package main

import "example.com/rtgtests/extended/multipackage/case030/pkg/a"
import "example.com/rtgtests/extended/multipackage/case030/pkg/b"

func main() {
	if a.Value()+b.Value() == 21 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
