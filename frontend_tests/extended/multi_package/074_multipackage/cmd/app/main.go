package main

import "example.com/renvotests/extended/multipackage/case074/pkg/a"
import "example.com/renvotests/extended/multipackage/case074/pkg/b"

func main() {
	if a.Value()+b.Value() == 25 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
