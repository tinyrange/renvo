package main

import "example.com/rtgtests/extended/multipackage/case084/pkg/a"
import "example.com/rtgtests/extended/multipackage/case084/pkg/b"

func main() {
	if a.Value()+b.Value() == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
