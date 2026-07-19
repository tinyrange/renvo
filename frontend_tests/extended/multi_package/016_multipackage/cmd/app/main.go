package main

import "example.com/renvotests/extended/multipackage/case016/pkg/a"
import "example.com/renvotests/extended/multipackage/case016/pkg/b"

func main() {
	if a.Value()+b.Value() == 35 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
