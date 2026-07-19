package main

import "example.com/renvotests/extended/multipackage/case005/pkg/a"
import "example.com/renvotests/extended/multipackage/case005/pkg/b"

func main() {
	if a.Value()+b.Value() == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
