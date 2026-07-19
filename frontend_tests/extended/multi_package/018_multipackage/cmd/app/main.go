package main

import "example.com/renvotests/extended/multipackage/case018/pkg/a"
import "example.com/renvotests/extended/multipackage/case018/pkg/b"

func main() {
	if a.Value()+b.Value() == 39 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
