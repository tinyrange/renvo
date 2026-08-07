package main

type compositeValue32Bit struct {
	total int64
	slots [2]int32
}

func compositeValue32BitSlot(value compositeValue32Bit) int32 {
	return value.slots[1]
}

func compositeValue32BitIdentity(value compositeValue32Bit) compositeValue32Bit {
	return value
}

func appMain(args []string) int {
	value := compositeValue32Bit{slots: [2]int32{0, 1}}
	if compositeValue32BitSlot(value) != 1 || compositeValue32BitIdentity(value).slots[1] != 1 {
		print("FAIL: composite value 32-bit call\n")
		return 1
	}
	print("PASS\n")
	return 0
}
