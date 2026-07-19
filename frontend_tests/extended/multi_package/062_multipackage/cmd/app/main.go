package main

import "example.com/renvotests/extended/multipackage/case062/pkg/a"
import "example.com/renvotests/extended/multipackage/case062/pkg/b"

func main() {
	if a.Value()+b.Value() == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
