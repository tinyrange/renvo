package main

import "example.com/renvotests/extended/multipackage/case115/pkg/a"
import "example.com/renvotests/extended/multipackage/case115/pkg/b"

func main() {
	if a.Value()+b.Value() == 4 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
