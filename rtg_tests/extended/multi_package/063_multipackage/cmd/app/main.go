package main

import "example.com/rtgtests/extended/multipackage/case063/pkg/a"
import "example.com/rtgtests/extended/multipackage/case063/pkg/b"

func main() {
	if a.Value()+b.Value() == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
