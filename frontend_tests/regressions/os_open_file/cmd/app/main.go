package main

import "os"

func main() {
	f, err := os.OpenFile("renvo-file-that-does-not-exist", os.O_RDONLY, 0)
	if err != nil {
		print("PASS\n")
		return
	}
	f.Close()
	print("FAIL\n")
}
