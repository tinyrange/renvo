package main

func promotedWideArgs(a, b, c, d, e, f, g, h uint64) uint64 {
	return (a ^ b) + (c ^ d) + (e ^ f) + (g ^ h)
}

func promotedWideMutate(value *uint64) { *value = *value ^ uint64(0xfedcba9876543210) }

func promotedWideLoop(value uint64, count int) uint64 {
	for i := 0; i < count; i++ {
		value = (value << 7) | (value >> 57)
		value = value ^ uint64(0x8000000100000081)
	}
	return value
}

func promotedWideFloat(a float64, count int) float64 {
	for i := 0; i < count; i++ {
		a = a*1.5 + 0.25
	}
	return a
}

func appMain(args []string) int {
	value := uint64(0x0123456789abcdef)
	// Independent 32-bit lane calculation checks whole-slot arithmetic and ABI
	// word accesses, including values live across nested calls and loop edges.
	lo, hi := uint32(value), uint32(value>>32)
	for i := 0; i < 97; i++ {
		lo, hi = (lo<<7)|(hi>>25), (hi<<7)|(lo>>25)
		lo = lo ^ uint32(0x00000081)
		hi = hi ^ uint32(0x80000001)
	}
	rotated := promotedWideLoop(value, 97)
	if uint32(rotated) != lo || uint32(rotated>>32) != hi {
		print("FAIL loop\n")
		return 1
	}
	if promotedWideArgs(0x100000001, 1, 0x200000002, 2, 0x400000004, 4,
		promotedWideLoop(0, 1), 0x8000000100000081) != uint64(0x700000000) {
		print("FAIL stack args\n")
		return 1
	}
	aliased := value
	promotedWideMutate(&aliased)
	if aliased != uint64(0xffffffffffffffff) {
		print("FAIL alias\n")
		return 1
	}
	var words [2]uint32
	words[0], words[1] = lo, hi
	if words[0] != uint32(rotated) || words[1] != uint32(rotated>>32) {
		print("FAIL aggregate\n")
		return 1
	}
	// Different primitive and aggregate lifetimes may reuse the same frame
	// ranges; addressed ranges must remain in memory for every lifetime.
	for i := 0; i < 4; i++ {
		if i&1 == 0 {
			x := uint64(i) + uint64(0x100000000)
			promotedWideMutate(&x)
			if x != (uint64(i)+uint64(0x100000000))^uint64(0xfedcba9876543210) {
				print("FAIL addressed lifetime\n")
				return 1
			}
		} else {
			x := float64(i) + 0.5
			if promotedWideFloat(x, 2) != (x*1.5+0.25)*1.5+0.25 {
				print("FAIL float lifetime\n")
				return 1
			}
		}
	}
	var signed int64 = -0x123456789abcdef
	shift := uint(70)
	if signed>>shift != -1 || uint64(signed)>>shift != 0 {
		print("FAIL large shift\n")
		return 1
	}
	print("PASS\n")
	return 0
}
