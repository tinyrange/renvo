package main

func renvo_runtime_CReadTimestampCounter(low *uint32, high *uint32)        {}
func renvo_runtime_CReadTimestampCounterOrdered(low *uint32, high *uint32) {}

func appMain(args []string) int {
	var low, high uint32
	renvo_runtime_CReadTimestampCounter(&low, &high)
	if low == 0 && high == 0 {
		return 1
	}
	var orderedLow, orderedHigh uint32
	renvo_runtime_CReadTimestampCounterOrdered(&orderedLow, &orderedHigh)
	if uint64(orderedHigh)<<32|uint64(orderedLow) < uint64(high)<<32|uint64(low) {
		return 2
	}
	print("PASS\n")
	return 0
}
