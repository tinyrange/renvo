package main

import "example.com/renvotests/extended/multipackage/case098/pkg/a"
import "example.com/renvotests/extended/multipackage/case098/pkg/b"

func main() {
	if a.Value()+b.Value() == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
