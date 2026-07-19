package main

import "example.com/renvotests/extended/multipackage/case017/pkg/a"
import "example.com/renvotests/extended/multipackage/case017/pkg/b"

func main() {
	if a.Value()+b.Value() == 37 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
