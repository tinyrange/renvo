package main

// Keep the arena above 8000h on compact 16-bit images. Arena bounds are
// addresses, so allocator overflow checks must remain unsigned there.
var arenaPointerUnsignedPadding [10000]byte

func appMain() int {
	arenaPointerUnsignedPadding[0] = 7
	persistent := new(int)
	*persistent = 33
	first := make([]byte, 1024)
	second := make([]byte, 1024)
	first[0] = 11
	second[len(second)-1] = 22
	if arenaPointerUnsignedPadding[0] != 7 || *persistent != 33 || first[0] != 11 || second[1023] != 22 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
