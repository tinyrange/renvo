package main

import "example.com/renvotests/extended/multipackage/case068/pkg/a"
import "example.com/renvotests/extended/multipackage/case068/pkg/b"

func main() {
	if a.Value()+b.Value() == 36 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
