package main

import "example.com/renvotests/extended/multipackage/case106/pkg/a"
import "example.com/renvotests/extended/multipackage/case106/pkg/b"

func main() {
	if a.Value()+b.Value() == 28 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
