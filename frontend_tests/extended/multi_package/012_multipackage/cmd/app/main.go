package main

import "example.com/renvotests/extended/multipackage/case012/pkg/a"
import "example.com/renvotests/extended/multipackage/case012/pkg/b"

func main() {
	if a.Value()+b.Value() == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
