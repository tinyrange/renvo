package main

func main() {
	if "abc"[1] == byte('b') {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
