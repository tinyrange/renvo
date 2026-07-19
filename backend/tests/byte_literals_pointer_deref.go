package main

func appMain(args []string) int {
	b := byte('S')
	p := &b
	if *p != 'S' {
		print("byte_literals_22 ptr\n")
		return 1
	}
	print("PASS\n")
	return 0
}
