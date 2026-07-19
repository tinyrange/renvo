package main

import "strings"

func main() {
	parts := strings.Split("a--b--", "--")
	if len(parts) == 3 && parts[0] == "a" && parts[1] == "b" && parts[2] == "" && strings.Join(parts, ":") == "a:b:" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
