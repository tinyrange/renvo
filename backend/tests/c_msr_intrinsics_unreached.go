package main

func renvo_runtime_CReadMSR(register uint32, low *uint32, high *uint32) {}
func renvo_runtime_CWriteMSR(register uint32, low uint32, high uint32)  {}
func renvo_runtime_CPause()                                             {}
func renvo_runtime_CReadCPUNODE(selector uint32) uint32                 { return 0 }
func renvo_runtime_CFarCall16(address uintptr)                          {}
func renvo_runtime_CDisableInterrupts()                                 {}
func renvo_runtime_CEnableInterrupts()                                  {}
func renvo_runtime_CLoadGDT(address uintptr)                            {}
func renvo_runtime_CLoadIDT(address uintptr)                            {}

func exerciseMSRIntrinsics(register uint32) {
	var low uint32
	var high uint32
	renvo_runtime_CReadMSR(register, &low, &high)
	renvo_runtime_CWriteMSR(register, low, high)
	renvo_runtime_CPause()
	low = renvo_runtime_CReadCPUNODE(low)
	renvo_runtime_CFarCall16(uintptr(low))
	renvo_runtime_CDisableInterrupts()
	renvo_runtime_CEnableInterrupts()
	renvo_runtime_CLoadGDT(uintptr(low))
	renvo_runtime_CLoadIDT(uintptr(low))
}

func appMain(args []string) int {
	if len(args) < 0 {
		exerciseMSRIntrinsics(0)
	}
	print("PASS\n")
	return 0
}
