package main

import "example.com/renvotests/extended/multipackage/case131/pkg/a"
import "example.com/renvotests/extended/multipackage/case131/pkg/b"

func main() {
	if a.Value()+b.Value() == 36 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
