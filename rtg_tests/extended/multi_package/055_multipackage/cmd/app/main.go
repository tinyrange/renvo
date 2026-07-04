package main

import "example.com/rtgtests/extended/multipackage/case055/pkg/a"
import "example.com/rtgtests/extended/multipackage/case055/pkg/b"

func main() {
	if a.Value()+b.Value() == 29 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
