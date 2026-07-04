package main

import "example.com/rtgtests/extended/multipackage/case096/pkg/a"
import "example.com/rtgtests/extended/multipackage/case096/pkg/b"

func main() {
	if a.Value()+b.Value() == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
