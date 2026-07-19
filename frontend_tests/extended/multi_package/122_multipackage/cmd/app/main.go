package main

import "example.com/renvotests/extended/multipackage/case122/pkg/a"
import "example.com/renvotests/extended/multipackage/case122/pkg/b"

func main() {
	if a.Value()+b.Value() == 18 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
