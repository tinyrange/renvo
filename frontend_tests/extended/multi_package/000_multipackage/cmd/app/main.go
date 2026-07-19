package main

import "example.com/renvotests/extended/multipackage/case000/pkg/a"
import "example.com/renvotests/extended/multipackage/case000/pkg/b"

func main() {
	if a.Value()+b.Value() == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
