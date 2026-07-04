package main

import "example.com/rtgtests/extended/multipackage/case038/pkg/a"
import "example.com/rtgtests/extended/multipackage/case038/pkg/b"

func main() {
	if a.Value()+b.Value() == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
