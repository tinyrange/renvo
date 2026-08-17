package main

func renvo_runtime_CAtomicFetchAdd8(address *uint64, value uint64) uint64  { return 0 }
func renvo_runtime_CAtomicFetchAdd16(address *uint64, value uint64) uint64 { return 0 }
func renvo_runtime_CAtomicFetchAdd32(address *uint64, value uint64) uint64 { return 0 }
func renvo_runtime_CAtomicFetchAdd64(address *uint64, value uint64) uint64 { return 0 }
func renvo_runtime_CAtomicExchange8(address *uint64, value uint64) uint64  { return 0 }
func renvo_runtime_CAtomicExchange16(address *uint64, value uint64) uint64 { return 0 }
func renvo_runtime_CAtomicExchange32(address *uint64, value uint64) uint64 { return 0 }
func renvo_runtime_CAtomicExchange64(address *uint64, value uint64) uint64 { return 0 }

func appMain(args []string) int {
	value := uint64(0x1122334455667788)
	address := &value
	if renvo_runtime_CAtomicFetchAdd8(address, 2) != 0x88 || value != 0x112233445566778a {
		return 1
	}
	if renvo_runtime_CAtomicFetchAdd16(address, 0x100) != 0x778a || value != 0x112233445566788a {
		return 2
	}
	if renvo_runtime_CAtomicFetchAdd32(address, 0x10000) != 0x5566788a || value != 0x112233445567788a {
		return 3
	}
	if renvo_runtime_CAtomicFetchAdd64(address, 0x100000000) != 0x112233445567788a || value != 0x112233455567788a {
		return 4
	}
	if renvo_runtime_CAtomicExchange8(address, 0xaa) != 0x8a || value != 0x11223345556778aa {
		return 5
	}
	if renvo_runtime_CAtomicExchange16(address, 0xbbcc) != 0x78aa || value != 0x112233455567bbcc {
		return 6
	}
	if renvo_runtime_CAtomicExchange32(address, 0xddeeff00) != 0x5567bbcc {
		return 71
	}
	if value != 0x11223345ddeeff00 {
		return 72
	}
	if renvo_runtime_CAtomicExchange64(address, 0x1020304050607080) != 0x11223345ddeeff00 || value != 0x1020304050607080 {
		return 8
	}
	print("PASS\n")
	return 0
}
