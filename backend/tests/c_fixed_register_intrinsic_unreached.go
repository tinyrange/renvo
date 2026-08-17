package main

func renvo_runtime_CLoadR9(value uint64) {}

func appMain(args []string) int {
	if len(args) == 99 {
		renvo_runtime_CLoadR9(0x1122334455667788)
	}
	print("PASS\n")
	return 0
}
