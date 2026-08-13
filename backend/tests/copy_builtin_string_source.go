package main

func main() {
	destination := []byte{'x', 'x', 'x', 'x', 'x'}
	if copy(destination[1:], "Go") != 2 || string(destination) != "xGoxx" {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
