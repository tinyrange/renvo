package main

import "example.com/renvotests/extended/multipackage/case123/pkg/a"
import "example.com/renvotests/extended/multipackage/case123/pkg/b"

func main() {
	if a.Value()+b.Value() == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
