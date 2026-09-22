package main

const globalShift = 2.0 << 3
const globalMask = ^uint16(0)
const globalUnsigned = ^uint32(0) >> 1

func appMain(args []string) int {
	const localShift = -2.0 >> 1
	if globalShift != 16 || globalMask != 65535 {
		return 1
	}
	if float64(globalShift)+0.5 != 16.5 {
		return 2
	}
	if float64(localShift)+0.5 != -0.5 {
		return 3
	}
	if float64(globalUnsigned) != 2147483647.0 {
		return 4
	}
	print("PASS\n")
	return 0
}
