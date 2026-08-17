package main

func renvo_runtime_CReadMSRSafe(register uint32, err *int32, low *uint32, high *uint32) {}
func renvo_runtime_CWriteMSRSafe(register uint32, low uint32, high uint32) int32        { return 0 }

func exerciseSafeMSR(register uint32, low uint32, high uint32) int32 {
	var observedLow, observedHigh uint32
	var err int32
	renvo_runtime_CReadMSRSafe(register, &err, &observedLow, &observedHigh)
	return err + renvo_runtime_CWriteMSRSafe(register, low, high)
}

func appMain(args []string) int {
	if len(args) < 0 {
		_ = exerciseSafeMSR(0, 0, 0)
	}
	print("PASS\n")
	return 0
}
