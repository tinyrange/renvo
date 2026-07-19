package main

import "example.com/renvotests/extended/multipackage/case015/pkg/a"
import "example.com/renvotests/extended/multipackage/case015/pkg/b"

func main() {
	if a.Value()+b.Value() == 33 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
