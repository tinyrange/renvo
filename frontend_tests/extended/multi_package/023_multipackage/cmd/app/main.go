package main

import "example.com/renvotests/extended/multipackage/case023/pkg/a"
import "example.com/renvotests/extended/multipackage/case023/pkg/b"

func main() {
	if a.Value()+b.Value() == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
