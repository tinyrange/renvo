package main

import "example.com/rtgtests/extended/multipackage/case022/pkg/a"
import "example.com/rtgtests/extended/multipackage/case022/pkg/b"

func main() {
	if a.Value()+b.Value() == 28 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
