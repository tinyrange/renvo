package main

func renvo_runtime_CInvalidatePage(address uintptr) {}

func appMain(args []string) int {
	if len(args) == 99 {
		renvo_runtime_CInvalidatePage(0x1000)
	}
	print("PASS\n")
	return 0
}
