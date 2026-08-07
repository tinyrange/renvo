package main

type wideCompoundBox struct {
	value uint64
}

func appMain(args []string) int {
	value := uint64(0x123456789abcdef0)
	value >>= 12
	value += uint64(0x100000001)
	if value != uint64(0x123466789abce) {
		print("FAIL: local wide compound assignment\n")
		return 1
	}
	value |= uint64(3)
	if value != uint64(0x123466789abcf) {
		if value == uint64(0x123466789abce) {
			print("FAIL: local wide bitwise compound assignment unchanged\n")
			return 1
		}
		print("FAIL: local wide bitwise compound assignment\n")
		return 1
	}
	values := []uint64{uint64(0x100000000)}
	if values[0] != uint64(0x100000000) {
		print("FAIL: indexed wide initial value\n")
		return 1
	}
	values[0] |= uint64(3)
	if values[0] != uint64(0x100000003) {
		if values[0] == uint64(0x100000000) {
			print("FAIL: indexed wide compound assignment unchanged\n")
			return 1
		}
		if values[0] == uint64(3) {
			print("FAIL: indexed wide compound assignment lost high word\n")
			return 1
		}
		print("FAIL: indexed wide compound assignment\n")
		return 1
	}
	box := wideCompoundBox{value: uint64(0x200000000)}
	box.value ^= uint64(7)
	if box.value != uint64(0x200000007) {
		print("FAIL: selected wide compound assignment\n")
		return 1
	}
	pointer := &box.value
	*pointer -= uint64(2)
	if box.value != uint64(0x200000005) {
		print("FAIL: indirect wide compound assignment\n")
		return 1
	}
	print("PASS\n")
	return 0
}
