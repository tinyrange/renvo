package main

import "example.com/renvotests/extended/multipackage/case079/pkg/a"
import "example.com/renvotests/extended/multipackage/case079/pkg/b"

func main() {
	if a.Value()+b.Value() == 16 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
