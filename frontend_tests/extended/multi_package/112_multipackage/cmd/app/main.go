package main

import "example.com/renvotests/extended/multipackage/case112/pkg/a"
import "example.com/renvotests/extended/multipackage/case112/pkg/b"

func main() {
	if a.Value()+b.Value() == 40 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
