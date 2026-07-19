package main

import "example.com/renvotests/extended/multipackage/case077/pkg/a"
import "example.com/renvotests/extended/multipackage/case077/pkg/b"

func main() {
	if a.Value()+b.Value() == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
