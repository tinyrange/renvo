package main

//renvo:load
func renvo_runtime_M11(address *uint8) uint8 {
	return *address
}

//renvo:store
func renvo_runtime_M12(address *uint8, value uint8) {
	*address = value
}

//renvo:load
func renvo_runtime_M23(address *uint16) uint16 {
	return *address
}

//renvo:store
func renvo_runtime_M24(address *uint16, value uint16) {
	*address = value
}

//renvo:load
func renvo_runtime_M45(address *uint32) uint32 {
	return *address
}

//renvo:store
func renvo_runtime_M46(address *uint32, value uint32) {
	*address = value
}

func taggedLoadThroughValue(load func(*uint32) uint32, address *uint32) uint32 {
	return load(address)
}

func appMain(args []string) int {
	value8 := uint8(0)
	value16 := uint16(0)
	value32 := uint32(0)
	renvo_runtime_M12(&value8, 0xa5)
	renvo_runtime_M24(&value16, 0x5aa5)
	renvo_runtime_M46(&value32, 41)
	if renvo_runtime_M11(&value8) != 0xa5 || renvo_runtime_M23(&value16) != 0x5aa5 || renvo_runtime_M45(&value32) != 41 || taggedLoadThroughValue(renvo_runtime_M45, &value32) != 41 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
