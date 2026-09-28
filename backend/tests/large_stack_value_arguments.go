package main

type stackValueBlock struct {
	data [259]byte
	tag  int
}

func stackValueMake(seed int) stackValueBlock {
	var value stackValueBlock
	for i := 0; i < len(value.data); i++ {
		value.data[i] = byte((seed + i*7) % 251)
	}
	value.tag = seed
	return value
}

func stackValueValid(value stackValueBlock, seed int) bool {
	if value.tag != seed {
		return false
	}
	for i := 0; i < len(value.data); i++ {
		if value.data[i] != byte((seed+i*7)%251) {
			return false
		}
	}
	return true
}

func stackValueMix(before int, first stackValueBlock, middle string, alias *stackValueBlock, second stackValueBlock, after int) stackValueBlock {
	if before != 19 || middle != "mixed arguments" || after != 73 || !stackValueValid(first, 11) || !stackValueValid(second, 23) {
		return stackValueBlock{}
	}
	// The aliased caller value and each by-value argument must remain distinct.
	alias.data[0] = 99
	alias.data[258] = 98
	alias.tag = 97
	if !stackValueValid(first, 11) || !stackValueValid(second, 23) {
		return stackValueBlock{}
	}
	first.data[0] = 91
	first.data[258] = 92
	first.tag = 93
	return first
}

func stackValueForward(first stackValueBlock, second stackValueBlock, alias *stackValueBlock) stackValueBlock {
	return stackValueMix(19, first, "mixed arguments", alias, second, 73)
}

func stackValuePair() (stackValueBlock, stackValueBlock) {
	return stackValueMake(31), stackValueMake(47)
}

func stackValueCheckPair(first stackValueBlock, second stackValueBlock) bool {
	return stackValueValid(first, 31) && stackValueValid(second, 47)
}

func appMain(args []string) int {
	first := stackValueMake(11)
	second := stackValueMake(23)
	result := stackValueForward(first, second, &first)
	if first.data[0] != 99 || first.data[258] != 98 || first.tag != 97 || !stackValueValid(second, 23) {
		print("FAIL caller values\n")
		return 1
	}
	if result.data[0] != 91 || result.data[258] != 92 || result.tag != 93 {
		print("FAIL returned value\n")
		return 1
	}
	for i := 1; i < 258; i++ {
		if result.data[i] != byte((11+i*7)%251) {
			print("FAIL argument body\n")
			return 1
		}
	}
	if !stackValueCheckPair(stackValuePair()) {
		print("FAIL tuple arguments\n")
		return 1
	}
	print("PASS\n")
	return 0
}
