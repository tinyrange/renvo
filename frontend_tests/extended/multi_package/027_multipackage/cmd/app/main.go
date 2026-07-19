package main

import "example.com/renvotests/extended/multipackage/case027/pkg/a"
import "example.com/renvotests/extended/multipackage/case027/pkg/b"

func main() {
	if a.Value()+b.Value() == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
