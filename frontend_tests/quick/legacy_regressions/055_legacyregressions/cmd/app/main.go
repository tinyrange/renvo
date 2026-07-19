package main

import "os"

func countEntries(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return -1
	}
	return len(entries)
}

func main() {
	if countEntries(".") > 0 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
