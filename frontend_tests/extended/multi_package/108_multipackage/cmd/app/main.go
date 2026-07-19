package main

import "example.com/renvotests/extended/multipackage/case108/pkg/a"
import "example.com/renvotests/extended/multipackage/case108/pkg/b"

func main() {
	if a.Value()+b.Value() == 32 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
