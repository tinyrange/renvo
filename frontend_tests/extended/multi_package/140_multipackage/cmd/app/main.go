package main

import "example.com/renvotests/extended/multipackage/case140/pkg/a"
import "example.com/renvotests/extended/multipackage/case140/pkg/b"

func main() {
	if a.Value()+b.Value() == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
