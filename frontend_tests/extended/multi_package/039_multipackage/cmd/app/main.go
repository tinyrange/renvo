package main

import "example.com/renvotests/extended/multipackage/case039/pkg/a"
import "example.com/renvotests/extended/multipackage/case039/pkg/b"

func main() {
	if a.Value()+b.Value() == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
