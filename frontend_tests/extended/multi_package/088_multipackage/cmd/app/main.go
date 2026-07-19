package main

import "example.com/renvotests/extended/multipackage/case088/pkg/a"
import "example.com/renvotests/extended/multipackage/case088/pkg/b"

func main() {
	if a.Value()+b.Value() == 34 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
