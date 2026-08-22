package main

func renvo_runtime_CWriteStartupSegments(value uint16) {}

func exerciseStartupSegmentsIntrinsic(value uint16) {
	renvo_runtime_CWriteStartupSegments(value)
}

func appMain(args []string) int {
	if len(args) < 0 {
		exerciseStartupSegmentsIntrinsic(24)
	}
	print("PASS\n")
	return 0
}
