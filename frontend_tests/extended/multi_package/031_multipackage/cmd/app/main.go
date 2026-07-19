package main

import "example.com/renvotests/extended/multipackage/case031/pkg/a"
import "example.com/renvotests/extended/multipackage/case031/pkg/b"

func main() {
	if a.Value()+b.Value() == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
