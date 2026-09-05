package main

func appMain(args []string) int {
	b := make([]byte, 3)
	if copy(b, "hello") != 3 || string(b) != "hel" { return 1 }
	if copy(b[:0], "x") != 0 || copy(b, "") != 0 { return 2 }
	print("PASS\n")
	return 0
}
