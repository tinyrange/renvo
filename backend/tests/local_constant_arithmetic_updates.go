package main

type arithmeticUpdateNamed int32

func updateSigned(value int32) int32 {
	value = value + 7
	value = value - 3
	value += 257
	value -= 256
	return value
}

func updateUnsigned(value uint32) uint32 {
	value = value + 7
	value = value - 3
	value += 257
	value -= 256
	return value
}

func updateNamed(value arithmeticUpdateNamed) arithmeticUpdateNamed {
	value = value + 7
	value = value - 3
	value += 257
	value -= 256
	return value
}

func appMain() int {
	if updateSigned(2147483645) != -2147483646 || updateUnsigned(4294967293) != 2 || updateNamed(2147483645) != -2147483646 {
		return 1
	}
	value := 23
	value = value + 9
	value += 4
	value -= 3
	if value != 33 {
		return 2
	}
	minimum := int32(0)
	minimum = minimum - -2147483648
	if minimum != -2147483648 {
		return 3
	}
	minimum = 0
	minimum -= -2147483648
	if minimum != -2147483648 {
		return 4
	}
	print("PASS\n")
	return 0
}
