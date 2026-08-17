package main

func renvo_runtime_CVerifySegment(address *uint16) {}

func appMain(args []string) int {
	selector := uint16(3 * 8)
	renvo_runtime_CVerifySegment(&selector)
	print("PASS\n")
	return 0
}
