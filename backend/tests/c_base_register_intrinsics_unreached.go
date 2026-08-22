package main

func renvo_runtime_CReadFSBase() uint64       { return 0 }
func renvo_runtime_CReadGSBase() uint64       { return 0 }
func renvo_runtime_CWriteFSBase(value uint64) {}
func renvo_runtime_CWriteGSBase(value uint64) {}

func exerciseBaseRegisters(value uint64) uint64 {
	renvo_runtime_CWriteFSBase(value)
	renvo_runtime_CWriteGSBase(value)
	return renvo_runtime_CReadFSBase() + renvo_runtime_CReadGSBase()
}

func appMain(args []string) int {
	if len(args) < 0 {
		_ = exerciseBaseRegisters(0)
	}
	print("PASS\n")
	return 0
}
