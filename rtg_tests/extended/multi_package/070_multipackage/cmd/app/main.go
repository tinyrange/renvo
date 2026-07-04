package main

import "example.com/rtgtests/extended/multipackage/case070/pkg/a"
import "example.com/rtgtests/extended/multipackage/case070/pkg/b"

func main() {
	if a.Value()+b.Value() == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
