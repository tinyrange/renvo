package main

import "example.com/rtgtests/extended/multipackage/case116/pkg/a"
import "example.com/rtgtests/extended/multipackage/case116/pkg/b"

func main() {
	if a.Value()+b.Value() == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
