package main

import "example.com/rtgtests/extended/multipackage/case040/pkg/a"
import "example.com/rtgtests/extended/multipackage/case040/pkg/b"

func main() {
	if a.Value()+b.Value() == 22 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
