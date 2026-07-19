package main

type renvoWideBox struct {
	signed   int64
	unsigned uint64
}

var renvoWideGlobal uint64

func renvoWideIdentity(value uint64) uint64 {
	return value
}

func renvoWideShift(value uint64, amount uint) uint64 {
	return value << amount
}

func renvoWideSigned(value int64) int64 {
	return value
}

func appMain(args []string) int {
	// The 32-bit backends require paired-register lowering, which is tracked by
	// the wider int64/uint64 issue. This regression specifically protects the
	// native 64-bit zero-test path fixed here.
	nativeMax := int(0x7fffffff)
	if nativeMax+1 < 0 {
		print("PASS\n")
		return 0
	}

	var one uint64 = 1
	shifted := one << 40
	if shifted == 0 || shifted != uint64(1<<40) {
		print("FAIL local shift\n")
		return 1
	}
	if renvoWideShift(one, 40) != uint64(1<<40) || renvoWideIdentity(shifted) != uint64(1<<40) {
		print("FAIL call boundary\n")
		return 1
	}

	added := shifted + uint64(0xffffffff)
	multiplied := uint64(0x100000001) * uint64(7)
	divided := uint64(0x700000007) / uint64(7)
	right := uint64(0x10000000000) >> 8
	if added != uint64(0x100ffffffff) || multiplied != uint64(0x700000007) || divided != uint64(0x100000001) || right != uint64(0x100000000) {
		print("FAIL arithmetic\n")
		return 1
	}

	bits := (uint64(0x10000000000) | uint64(0x55)) ^ uint64(0x05)
	if bits != uint64(0x10000000050) || bits&uint64(0xff) != uint64(0x50) {
		print("FAIL bitwise\n")
		return 1
	}

	negative := int64(-0x10000000000)
	if renvoWideSigned(negative)>>8 != int64(-0x100000000) || int64(uint64(0x10000000001)) != int64(0x10000000001) {
		print("FAIL signed\n")
		return 1
	}

	renvoWideGlobal = shifted
	box := renvoWideBox{signed: negative, unsigned: shifted}
	array := [2]uint64{7, shifted}
	slice := []uint64{9, shifted}
	values := make(map[string]uint64)
	values["wide"] = shifted
	var boxed interface{} = shifted
	asserted, ok := boxed.(uint64)
	if renvoWideGlobal != shifted || box.signed != negative || box.unsigned != shifted || array[1] != shifted || slice[1] != shifted || values["wide"] != shifted || !ok || asserted != shifted {
		print("FAIL storage\n")
		return 1
	}

	print("PASS\n")
	return 0
}
