package main

import "example.com/rtgtests/extended/multipackage/case045/pkg/a"
import "example.com/rtgtests/extended/multipackage/case045/pkg/b"

func main() {
	if a.Value()+b.Value() == 32 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
