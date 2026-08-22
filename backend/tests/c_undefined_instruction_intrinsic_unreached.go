package main

func renvo_runtime_CUndefinedInstruction() {}

func appMain(args []string) int {
	if len(args) < 0 {
		renvo_runtime_CUndefinedInstruction()
	}
	print("PASS\n")
	return 0
}
