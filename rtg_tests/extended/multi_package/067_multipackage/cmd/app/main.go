package main

import "example.com/rtgtests/extended/multipackage/case067/pkg/a"
import "example.com/rtgtests/extended/multipackage/case067/pkg/b"

func main() {
	if a.Value()+b.Value() == 34 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
