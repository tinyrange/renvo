package main

import "strings"

func main() {
	if strings.Count("abc", "") == 4 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
