package main

func renvo_runtime_CNonTemporalStore32(address *uint32, value uint64) {}
func renvo_runtime_CNonTemporalStore64(address *uint64, value uint64) {}

func appMain(args []string) int {
	value32 := uint32(0)
	value64 := uint64(0)
	renvo_runtime_CNonTemporalStore32(&value32, 0x12345678)
	renvo_runtime_CNonTemporalStore64(&value64, 0x123456789abcdef0)
	if value32 != 0x12345678 || value64 != 0x123456789abcdef0 {
		return 1
	}
	print("PASS\n")
	return 0
}
