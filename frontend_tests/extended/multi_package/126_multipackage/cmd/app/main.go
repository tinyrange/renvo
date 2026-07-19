package main

import "example.com/renvotests/extended/multipackage/case126/pkg/a"
import "example.com/renvotests/extended/multipackage/case126/pkg/b"

func main() {
	if a.Value()+b.Value() == 26 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
