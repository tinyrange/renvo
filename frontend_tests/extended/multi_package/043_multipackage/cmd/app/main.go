package main

import "example.com/renvotests/extended/multipackage/case043/pkg/a"
import "example.com/renvotests/extended/multipackage/case043/pkg/b"

func main() {
	if a.Value()+b.Value() == 28 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
