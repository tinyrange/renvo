package main

func renvo_runtime_CLoadLDT(value uint16) {}
func renvo_runtime_CStoreLDT() uint64     { return 0 }

func exerciseLDTIntrinsics(value uint16) uint64 {
	renvo_runtime_CLoadLDT(value)
	return renvo_runtime_CStoreLDT()
}

func appMain(args []string) int {
	if len(args) < 0 {
		_ = exerciseLDTIntrinsics(1)
	}
	print("PASS\n")
	return 0
}
