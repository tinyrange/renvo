package main

import "example.com/renvotests/extended/multipackage/case144/pkg/a"
import "example.com/renvotests/extended/multipackage/case144/pkg/b"

func main() {
	if a.Value()+b.Value() == 20 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
