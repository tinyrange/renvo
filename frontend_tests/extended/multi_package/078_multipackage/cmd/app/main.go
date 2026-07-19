package main

import "example.com/renvotests/extended/multipackage/case078/pkg/a"
import "example.com/renvotests/extended/multipackage/case078/pkg/b"

func main() {
	if a.Value()+b.Value() == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
