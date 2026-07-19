package main

import "example.com/renvotests/extended/multipackage/case065/pkg/a"
import "example.com/renvotests/extended/multipackage/case065/pkg/b"

func main() {
	if a.Value()+b.Value() == 30 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
