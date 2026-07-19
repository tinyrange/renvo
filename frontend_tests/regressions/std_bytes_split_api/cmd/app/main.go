package main

import "bytes"

func main() {
	parts := bytes.Split([]byte("a,b,c"), []byte(","))
	if len(parts) == 3 && string(bytes.Join(parts, []byte("|"))) == "a|b|c" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
