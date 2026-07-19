package main

import "example.com/renvotests/extended/multipackage/case034/pkg/a"
import "example.com/renvotests/extended/multipackage/case034/pkg/b"

func main() {
	if a.Value()+b.Value() == 29 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
