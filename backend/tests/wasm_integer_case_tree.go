package main

type dispatchWord uint32

const dispatchHigh dispatchWord = 0x80000000

var integerEffects int

func signedCases(n int32) int {
	switch n {
	case -2147483648:
		return 1
	case 17:
		return 2
	case -1000000:
		return 3
	case 2147483647:
		return 4
	case 0:
		return 5
	case -1:
		return 6
	case 1024:
		return 7
	case -2147483647:
		return 8
	case 3:
		return 9
	case 2147483646:
		return 10
	case -100:
		return 11
	case 1:
		return 12
	default:
		return -1
	}
}

func unsignedCases(n dispatchWord) int {
	switch n {
	case 0:
		return 1
	case 4294967295:
		return 2
	case 1:
		return 3
	case dispatchHigh:
		return 4
	case 127:
		return 5
	case 3735928559:
		return 6
	case 2147483647:
		return 7
	case 65535:
		return 8
	case 2147483649:
		return 9
	case 4294967294:
		return 10
	case 128:
		return 11
	case 305419896:
		return 12
	default:
		return -1
	}
}

func byteCases(n int8) int {
	switch n {
	case -128:
		return 1
	case 50:
		return 2
	case -127:
		return 3
	case 126:
		return 4
	case -1:
		return 5
	case 0:
		return 6
	case 127:
		return 7
	case -50:
		return 8
	case 1:
		return 9
	default:
		return -1
	}
}

func halfCases(n uint16) int {
	switch n {
	case 65535:
		return 1
	case 0:
		return 2
	case 32768:
		return 3
	case 1:
		return 4
	case 32767:
		return 5
	case 127:
		return 6
	case 32769:
		return 7
	case 255:
		return 8
	case 65534:
		return 9
	default:
		return -1
	}
}

func effectInteger(n int32) int32 { integerEffects++; return n }
func orderedInteger(n int32) int {
	switch effectInteger(n) {
	case effectInteger(-7):
		return 1
	case -4, effectInteger(3):
		return 2
	case 17:
		return 3
	case 21:
		return 4
	case effectInteger(100):
		return 5
	case 200:
		return 6
	case 300:
		return 7
	default:
		return 8
	}
}
func fallInteger(n int32) int {
	result := 0
	switch n {
	case -2147483648:
		result += 1
		fallthrough
	default:
		result += 2
	case 2147483647, 17:
		result += 4
		fallthrough
	case -1:
		result += 8
	case 0:
		result += 16
	case 1:
		result += 32
	case -2147483647:
		result += 64
	}
	return result
}
func appMain(args []string) int {
	signed := []int32{-2147483648, 17, -1000000, 2147483647, 0, -1, 1024, -2147483647, 3, 2147483646, -100, 1}
	unsigned := []dispatchWord{0, 4294967295, 1, 2147483648, 127, 3735928559, 2147483647, 65535, 2147483649, 4294967294, 128, 305419896}
	for i := 0; i < len(signed); i++ {
		if signedCases(signed[i]) != i+1 {
			print("FAIL signed hit\n")
			return 1
		}
		for delta := -2; delta <= 2; delta++ {
			changed := signed[i] + int32(delta)
			want := -1
			for j := 0; j < len(signed); j++ {
				if signed[j] == changed {
					want = j + 1
					break
				}
			}
			if signedCases(changed) != want {
				print("FAIL signed miss\n")
				return 1
			}
		}
	}
	for i := 0; i < len(unsigned); i++ {
		if unsignedCases(unsigned[i]) != i+1 {
			print("FAIL unsigned hit\n")
			return 1
		}
		for delta := -2; delta <= 2; delta++ {
			changed := unsigned[i] + dispatchWord(delta)
			want := -1
			for j := 0; j < len(unsigned); j++ {
				if unsigned[j] == changed {
					want = j + 1
					break
				}
			}
			if unsignedCases(changed) != want {
				print("FAIL unsigned miss\n")
				return 1
			}
		}
	}
	bytes := []int8{-128, 50, -127, 126, -1, 0, 127, -50, 1}
	for i := 0; i < len(bytes); i++ {
		if byteCases(bytes[i]) != i+1 {
			print("FAIL signed byte\n")
			return 1
		}
	}
	halves := []uint16{65535, 0, 32768, 1, 32767, 127, 32769, 255, 65534}
	for i := 0; i < len(halves); i++ {
		if halfCases(halves[i]) != i+1 {
			print("FAIL unsigned half\n")
			return 1
		}
	}
	if byteCases(5) != -1 || halfCases(400) != -1 {
		print("FAIL narrow miss\n")
		return 1
	}
	integerEffects = 0
	if orderedInteger(-4) != 2 || integerEffects != 2 {
		print("FAIL ordered early\n")
		return 1
	}
	integerEffects = 0
	if orderedInteger(3) != 2 || integerEffects != 3 {
		print("FAIL ordered late\n")
		return 1
	}
	integerEffects = 0
	if orderedInteger(900) != 8 || integerEffects != 4 {
		print("FAIL ordered default\n")
		return 1
	}
	if fallInteger(-2147483648) != 3 || fallInteger(17) != 12 || fallInteger(2147483647) != 12 || fallInteger(-1) != 8 || fallInteger(-2147483647) != 64 || fallInteger(24) != 2 {
		print("FAIL integer fallthrough\n")
		return 1
	}
	integerEffects = 0
	if signedCases(effectInteger(17)) != 2 || integerEffects != 1 {
		print("FAIL integer once\n")
		return 1
	}
	print("PASS\n")
	return 0
}
