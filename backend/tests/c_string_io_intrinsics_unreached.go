package main

func renvo_runtime_CPortInString8(port uint16, address uintptr, count uintptr)   {}
func renvo_runtime_CPortInString16(port uint16, address uintptr, count uintptr)  {}
func renvo_runtime_CPortInString32(port uint16, address uintptr, count uintptr)  {}
func renvo_runtime_CPortOutString8(port uint16, address uintptr, count uintptr)  {}
func renvo_runtime_CPortOutString16(port uint16, address uintptr, count uintptr) {}
func renvo_runtime_CPortOutString32(port uint16, address uintptr, count uintptr) {}

func exerciseStringIO(address uintptr) {
	renvo_runtime_CPortInString8(1, address, 2)
	renvo_runtime_CPortInString16(1, address, 2)
	renvo_runtime_CPortInString32(1, address, 2)
	renvo_runtime_CPortOutString8(1, address, 2)
	renvo_runtime_CPortOutString16(1, address, 2)
	renvo_runtime_CPortOutString32(1, address, 2)
}

func appMain(args []string) int {
	if len(args) < 0 {
		exerciseStringIO(0)
	}
	print("PASS\n")
	return 0
}
