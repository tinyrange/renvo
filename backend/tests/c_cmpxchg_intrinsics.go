package main

func renvo_runtime_CCompareExchange8(address *uint8, observed *uint8, replacement uint64) uint8 {
	return 0
}
func renvo_runtime_CCompareExchange16(address *uint16, observed *uint16, replacement uint64) uint8 {
	return 0
}
func renvo_runtime_CCompareExchange32(address *uint32, observed *uint32, replacement uint64) uint8 {
	return 0
}
func renvo_runtime_CCompareExchange64(address *uint64, observed *uint64, replacement uint64) uint8 {
	return 0
}

func appMain(args []string) int {
	value8, old8 := uint8(3), uint8(3)
	if renvo_runtime_CCompareExchange8(&value8, &old8, 5) != 1 || value8 != 5 || old8 != 3 {
		return 1
	}
	value16, old16 := uint16(7), uint16(9)
	if renvo_runtime_CCompareExchange16(&value16, &old16, 11) != 0 || value16 != 7 || old16 != 7 {
		return 2
	}
	value32, old32 := uint32(13), uint32(13)
	if renvo_runtime_CCompareExchange32(&value32, &old32, 17) != 1 || value32 != 17 || old32 != 13 {
		return 3
	}
	value64, old64 := uint64(1<<40), uint64(19)
	if renvo_runtime_CCompareExchange64(&value64, &old64, 23) != 0 || value64 != 1<<40 || old64 != 1<<40 {
		return 4
	}
	print("PASS\n")
	return 0
}
