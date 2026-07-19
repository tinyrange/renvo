package main

import "example.com/renvotests/extended/multipackage/case083/pkg/a"
import "example.com/renvotests/extended/multipackage/case083/pkg/b"

func main() {
	if a.Value()+b.Value() == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
