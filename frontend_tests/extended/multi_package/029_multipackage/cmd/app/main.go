package main

import "example.com/renvotests/extended/multipackage/case029/pkg/a"
import "example.com/renvotests/extended/multipackage/case029/pkg/b"

func main() {
	if a.Value()+b.Value() == 19 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
