package main

import "example.com/renvotests/extended/multipackage/case090/pkg/a"
import "example.com/renvotests/extended/multipackage/case090/pkg/b"

func main() {
	if a.Value()+b.Value() == 38 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
