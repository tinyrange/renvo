package main

func renvo0595Mutate(bs []byte) {
	if len(bs) > 0 && bs[0] == 'h' {
		bs[0] = 'H'
	}
}

func appMain(args []string) int {
	bs := []byte("hi")
	renvo0595Mutate(bs)
	if bs[0] != 'H' {
		print("RENVO-0595 helper mutation failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
