package main

func appMain(args []string) int {
	bs := []byte("cat")
	goto mutate
mutate:
	bs[1] = 'o'
	if bs[0] != 'c' || bs[1] != 'o' || bs[2] != 't' {
		print("RENVO-0562 byte index assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
