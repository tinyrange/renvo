package main

import "example.com/renvotests/extended/multipackage/case146/pkg/a"
import "example.com/renvotests/extended/multipackage/case146/pkg/b"

func main() {
	if a.Value()+b.Value() == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
