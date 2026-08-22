package main

var portableInstructionPosition uint64 = 1

func renvo_runtime_CReadInstructionPointer() uint64 { return portableInstructionPosition }

func appMain(args []string) int {
	if renvo_runtime_CReadInstructionPointer() == 0 {
		return 1
	}
	print("PASS\n")
	return 0
}
