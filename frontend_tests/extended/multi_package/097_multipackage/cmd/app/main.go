package main

import "example.com/renvotests/extended/multipackage/case097/pkg/a"
import "example.com/renvotests/extended/multipackage/case097/pkg/b"

func main() {
	if a.Value()+b.Value() == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
