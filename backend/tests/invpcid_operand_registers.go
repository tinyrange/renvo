package main

func renvo_runtime_CInvalidateProcessContext(address uintptr, kind uintptr) {}

func issueInvalidateProcessContext(address uintptr, kind uintptr) {
	renvo_runtime_CInvalidateProcessContext(address, kind)
}

func appMain(args []string) int {
	if len(args) > 1000 {
		issueInvalidateProcessContext(0, 2)
	}
	print("PASS\n")
	return 0
}
