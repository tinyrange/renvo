package main

import "strings"

func main() {
	parts := strings.Split("abc", ",")
	if len(parts) == 1 && parts[0] == "abc" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
