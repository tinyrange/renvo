package main

func renvo_runtime_CMemoryReadFence() {}
func renvo_runtime_CMemoryFence()     {}
func renvo_runtime_CReadFence()       {}
func renvo_runtime_CWriteFence()      {}

func appMain(args []string) int {
	renvo_runtime_CMemoryReadFence()
	renvo_runtime_CMemoryFence()
	renvo_runtime_CReadFence()
	renvo_runtime_CWriteFence()
	print("PASS\n")
	return 0
}
