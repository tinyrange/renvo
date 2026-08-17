package main

func renvo_runtime_CReadInstructionPointer() uint64 { return 0 }

func appMain(args []string) int {
	if renvo_runtime_CReadInstructionPointer() == 0 {
		return 1
	}
	print("PASS\n")
	return 0
}
