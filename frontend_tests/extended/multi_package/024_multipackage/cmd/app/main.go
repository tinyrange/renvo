package main

import "example.com/renvotests/extended/multipackage/case024/pkg/a"
import "example.com/renvotests/extended/multipackage/case024/pkg/b"

func main() {
	if a.Value()+b.Value() == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
