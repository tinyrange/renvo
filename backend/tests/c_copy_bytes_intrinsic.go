package main

func renvo_runtime_CCopyBytes(dest *byte, src *byte, count uintptr) {}

func appMain(args []string) int {
	source := [5]byte{'P', 'A', 'S', 'S', '\n'}
	var destination [5]byte
	renvo_runtime_CCopyBytes(&destination[0], &source[0], 5)
	if destination[0] == 'P' && destination[3] == 'S' && destination[4] == '\n' {
		print("PASS\n")
	}
	return 0
}
