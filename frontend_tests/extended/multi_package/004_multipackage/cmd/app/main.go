package main

import "example.com/renvotests/extended/multipackage/case004/pkg/a"
import "example.com/renvotests/extended/multipackage/case004/pkg/b"

func main() {
	corpusOK := false
	if a.Value()+b.Value() == 11 {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
