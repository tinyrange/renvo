package main

func renvo_runtime_CCopyBytes(dest *[256]byte, src *[256]byte, count uintptr) {
	for i := uintptr(0); i < count; i++ {
		dest[i] = src[i]
	}
}

func appMain(args []string) int {
	source := [256]byte{'P', 'A', 'S', 'S', '\n'}
	var destination [256]byte
	renvo_runtime_CCopyBytes(&destination, &source, 5)
	if destination[0] == 'P' && destination[3] == 'S' && destination[4] == '\n' {
		print("PASS\n")
	}
	return 0
}
