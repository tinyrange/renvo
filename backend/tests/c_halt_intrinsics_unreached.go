package main

func renvo_runtime_CHalt()                     {}
func renvo_runtime_CEnableInterruptsAndHalt()  {}
func renvo_runtime_CWriteBackInvalidateCache() {}
func renvo_runtime_CSwapGS()                   {}

func exerciseHaltIntrinsics() {
	renvo_runtime_CHalt()
	renvo_runtime_CEnableInterruptsAndHalt()
	renvo_runtime_CWriteBackInvalidateCache()
	renvo_runtime_CSwapGS()
}

func appMain(args []string) int {
	if len(args) < 0 {
		exerciseHaltIntrinsics()
	}
	print("PASS\n")
	return 0
}
