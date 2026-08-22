package main

func subwordFlags(value uint64, sign uint64) uint8 {
	flags := uint8(0)
	if value == 0 {
		flags = flags | 1
	}
	if value&sign != 0 {
		flags = flags | 2
	}
	return flags
}

func renvo_runtime_CAtomicAdd8(address *uint64, value uint32) uint8 {
	result := (*address + uint64(value)) & 0xff
	*address = *address&^0xff | result
	return subwordFlags(result, 0x80)
}
func renvo_runtime_CAtomicAdd16(address *uint64, value uint32) uint8 {
	result := (*address + uint64(value)) & 0xffff
	*address = *address&^0xffff | result
	return subwordFlags(result, 0x8000)
}
func renvo_runtime_CAtomicInc8(address *uint64, value uint32) uint8 {
	result := (*address + 1) & 0xff
	*address = *address&^0xff | result
	return subwordFlags(result, 0x80)
}
func renvo_runtime_CAtomicDec16(address *uint64, value uint32) uint8 {
	result := (*address - 1) & 0xffff
	*address = *address&^0xffff | result
	return subwordFlags(result, 0x8000)
}

func appMain(args []string) int {
	value := uint64(0x1122334455667788)
	address := &value
	renvo_runtime_CAtomicAdd8(address, 2)
	renvo_runtime_CAtomicAdd16(address, 0x100)
	renvo_runtime_CAtomicInc8(address, 1)
	renvo_runtime_CAtomicDec16(address, 1)
	if value != 0x112233445566788a {
		return 1
	}
	print("PASS\n")
	return 0
}
