package main

func appMain(args []string) int {
	b := 'H'
	if int(b) != 72 {
		print("byte_literals_13 int\n")
		return 1
	}
	print("PASS\n")
	return 0
}
