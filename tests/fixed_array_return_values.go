package main

func fixedArrayRows(value int) [2]byte {
	if value == 1 {
		return [2]byte{3, 4}
	}
	return [2]byte{}
}

func fixedArrayRelay(value int) [2]byte {
	if value == 2 {
		local := [2]byte{5, 6}
		return local
	}
	return fixedArrayRows(value)
}

func fixedArraySum(value [2]byte) int {
	return int(value[0]) + int(value[1])
}

func appMain(args []string) int {
	first := fixedArrayRows(1)
	empty := fixedArrayRows(0)
	local := fixedArrayRelay(2)
	forwarded := fixedArrayRelay(1)
	if first[0] == 3 && first[1] == 4 &&
		empty[0] == 0 && empty[1] == 0 &&
		local[0] == 5 && local[1] == 6 &&
		forwarded[0] == 3 && forwarded[1] == 4 &&
		fixedArrayRows(1)[1] == 4 &&
		fixedArraySum(fixedArrayRows(1)) == 7 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
