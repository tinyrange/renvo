package main

func renvo_runtime_CReadPerformanceCounter(counter uint32, low *uint32, high *uint32) {}

func appMain(args []string) int {
	if len(args) < 0 {
		var low, high uint32
		renvo_runtime_CReadPerformanceCounter(0, &low, &high)
	}
	print("PASS\n")
	return 0
}
