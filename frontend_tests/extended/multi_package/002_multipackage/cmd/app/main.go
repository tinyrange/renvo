package main

import "example.com/renvotests/extended/multipackage/case002/pkg/a"
import "example.com/renvotests/extended/multipackage/case002/pkg/b"

func main() {
	if a.Value()+b.Value() == 7 {
		print("PASS\n")
		return
	} else {

		print("FAIL\n")
	}
}
