package main

// renvo:module-license GPL

// renvo:linkstatic kernel,for_each_kernel_tracepoint
func kernelForEachTracepoint(callback func(uintptr, uintptr), data uintptr) {}

func visitTracepoint(tracepoint uintptr, data uintptr) {
	if tracepoint == data {
		print("")
	}
}

func appMain() {
	kernelForEachTracepoint(visitTracepoint, 0)
	print("PREPARED_KERNEL_PASS\n")
}

func moduleExit() {
	kernelForEachTracepoint(visitTracepoint, 0)
}
