package main

import "strings"

func main() {
	parts := strings.Split("abc", "")
	if len(parts) == 3 && parts[0] == "a" && parts[1] == "b" && parts[2] == "c" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
