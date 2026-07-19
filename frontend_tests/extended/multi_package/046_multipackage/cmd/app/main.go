package main

import "example.com/renvotests/extended/multipackage/case046/pkg/a"
import "example.com/renvotests/extended/multipackage/case046/pkg/b"

func main() {
	if a.Value()+b.Value() == 11 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
