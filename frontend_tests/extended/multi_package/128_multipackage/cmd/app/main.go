package main

import "example.com/renvotests/extended/multipackage/case128/pkg/a"
import "example.com/renvotests/extended/multipackage/case128/pkg/b"

func main() {
	if a.Value()+b.Value() == 30 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
