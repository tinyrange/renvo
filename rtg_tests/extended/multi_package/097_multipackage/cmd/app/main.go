package main

import "example.com/rtgtests/extended/multipackage/case097/pkg/a"
import "example.com/rtgtests/extended/multipackage/case097/pkg/b"

func main() {
	if a.Value()+b.Value() == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
