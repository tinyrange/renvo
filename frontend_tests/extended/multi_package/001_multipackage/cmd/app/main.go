package main

import "example.com/renvotests/extended/multipackage/case001/pkg/a"
import "example.com/renvotests/extended/multipackage/case001/pkg/b"

func main() {
	corpusOK := a.Value()+b.Value() == 5
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
