package main

import "example.com/renvotests/extended/multipackage/case117/pkg/a"
import "example.com/renvotests/extended/multipackage/case117/pkg/b"

func main() {
	if a.Value()+b.Value() == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
