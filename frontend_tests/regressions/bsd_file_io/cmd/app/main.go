package main

import "os"

func main() {
	name := "/tmp/renvo-bsd-file-io.tmp"
	if os.WriteFile(name, []byte("bsd"), 0600) != nil {
		print("FAIL\n")
		return
	}
	got, err := os.ReadFile(name)
	if err != nil || string(got) != "bsd" {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
