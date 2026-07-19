package main

import "example.com/renvotests/extended/multipackage/case137/pkg/a"
import "example.com/renvotests/extended/multipackage/case137/pkg/b"

func main() {
	if a.Value()+b.Value() == 29 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
