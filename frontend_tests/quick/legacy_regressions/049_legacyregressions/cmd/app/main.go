package main

func main() {
	if "abcdef"[1:4] == "bcd" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
