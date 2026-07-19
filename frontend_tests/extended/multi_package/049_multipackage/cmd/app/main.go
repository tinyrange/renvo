package main

import "example.com/renvotests/extended/multipackage/case049/pkg/a"
import "example.com/renvotests/extended/multipackage/case049/pkg/b"

func main() {
	if a.Value()+b.Value() == 17 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
