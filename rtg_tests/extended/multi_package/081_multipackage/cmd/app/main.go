package main

import "example.com/rtgtests/extended/multipackage/case081/pkg/a"
import "example.com/rtgtests/extended/multipackage/case081/pkg/b"

func main() {
	if a.Value()+b.Value() == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
