package main

func renvo_runtime_CAtomicAdd8(address *uint64, value uint64) uint8  { return 0 }
func renvo_runtime_CAtomicAdd16(address *uint64, value uint64) uint8 { return 0 }
func renvo_runtime_CAtomicInc8(address *uint64, value uint64) uint8  { return 0 }
func renvo_runtime_CAtomicDec16(address *uint64, value uint64) uint8 { return 0 }

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
