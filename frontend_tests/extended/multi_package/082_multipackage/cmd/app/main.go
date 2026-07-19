package main

import "example.com/renvotests/extended/multipackage/case082/pkg/a"
import "example.com/renvotests/extended/multipackage/case082/pkg/b"

func main() {
	if a.Value()+b.Value() == 22 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
