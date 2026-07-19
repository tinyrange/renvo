package main

import "bytes"

func main() {
	fromString := bytes.NewBufferString("ab")
	fromBytes := bytes.NewBuffer([]byte("cd"))
	if fromString.String() == "ab" && fromString.Len() == 2 && fromBytes.String() == "cd" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
