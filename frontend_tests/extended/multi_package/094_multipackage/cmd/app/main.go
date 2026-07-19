package main

import "example.com/renvotests/extended/multipackage/case094/pkg/a"
import "example.com/renvotests/extended/multipackage/case094/pkg/b"

func main() {
	if a.Value()+b.Value() == 23 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
