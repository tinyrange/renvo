package main

import "example.com/rtgtests/extended/multipackage/case110/pkg/a"
import "example.com/rtgtests/extended/multipackage/case110/pkg/b"

func main() {
	if a.Value()+b.Value() == 36 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
