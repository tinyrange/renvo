package main

import "example.com/renvotests/extended/multipackage/case009/pkg/a"
import "example.com/renvotests/extended/multipackage/case009/pkg/b"

func main() {
	if a.Value()+b.Value() == 21 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
