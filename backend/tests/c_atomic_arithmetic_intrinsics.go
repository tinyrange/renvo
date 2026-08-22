package main

func atomicFlags32(value int32) uint8 {
	flags := uint8(0)
	if value == 0 {
		flags = flags | 1
	}
	if value < 0 {
		flags = flags | 2
	}
	return flags
}

func atomicFlags64(value int64) uint8 {
	flags := uint8(0)
	if value == 0 {
		flags = flags | 1
	}
	if value < 0 {
		flags = flags | 2
	}
	return flags
}

func renvo_runtime_CAtomicAdd32(address *int32, value uint32) uint8 {
	*address = *address + int32(value)
	return atomicFlags32(*address)
}
func renvo_runtime_CAtomicSub32(address *int32, value uint32) uint8 {
	*address = *address - int32(value)
	return atomicFlags32(*address)
}
func renvo_runtime_CAtomicAnd32(address *int32, value uint32) uint8 {
	*address = *address & int32(value)
	return atomicFlags32(*address)
}
func renvo_runtime_CAtomicOr32(address *int32, value uint32) uint8 {
	*address = *address | int32(value)
	return atomicFlags32(*address)
}
func renvo_runtime_CAtomicXor32(address *int32, value uint32) uint8 {
	*address = *address ^ int32(value)
	return atomicFlags32(*address)
}
func renvo_runtime_CAtomicInc32(address *int32, value uint32) uint8 {
	*address = *address + 1
	return atomicFlags32(*address)
}
func renvo_runtime_CAtomicDec32(address *int32, value uint32) uint8 {
	*address = *address - 1
	return atomicFlags32(*address)
}
func renvo_runtime_CAtomicAdd64(address *int64, value uint64) uint8 {
	*address = *address + int64(value)
	return atomicFlags64(*address)
}
func renvo_runtime_CAtomicSub64(address *int64, value uint64) uint8 {
	*address = *address - int64(value)
	return atomicFlags64(*address)
}

func appMain(args []string) int {
	value32 := int32(3)
	address32 := &value32
	renvo_runtime_CAtomicAdd32(address32, 5)
	renvo_runtime_CAtomicSub32(address32, 2)
	renvo_runtime_CAtomicAnd32(address32, 7)
	renvo_runtime_CAtomicOr32(address32, 8)
	renvo_runtime_CAtomicXor32(address32, 3)
	renvo_runtime_CAtomicInc32(address32, 1)
	if renvo_runtime_CAtomicDec32(address32, 1)&1 != 0 || value32 != 13 {
		return 1
	}
	value32 = 1
	if renvo_runtime_CAtomicSub32(address32, 1)&1 != 1 || value32 != 0 {
		return 2
	}
	value32 = 0
	negativeFlags := renvo_runtime_CAtomicSub32(address32, 1)
	if (negativeFlags>>1)&1 != 1 {
		return 3
	}
	if value32+1 != 0 {
		return 5
	}
	value64 := int64(1 << 40)
	address64 := &value64
	renvo_runtime_CAtomicAdd64(address64, 9)
	renvo_runtime_CAtomicSub64(address64, 4)
	if value64 != 1<<40+5 {
		return 4
	}
	print("PASS\n")
	return 0
}
