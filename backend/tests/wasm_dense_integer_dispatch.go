package main

type denseWord uint32

var denseEffects int

func denseEffect(n int32) int32 { denseEffects++; return n }
func denseSigned(n int32) int {
	switch n {
	case 5:
		return 105
	case -2, 0:
		return 100
	default:
		return -1
	case 7:
		return 107
	case -1:
		return 99
	case 2:
		return 102
	case 4:
		return 104
	case 6:
		return 106
	}
}
func denseLow(n int32) int {
	switch n {
	case -2147483648:
		return 1
	case -2147483647:
		return 2
	case -2147483646:
		return 3
	case -2147483644:
		return 5
	case -2147483643:
		return 6
	case -2147483642:
		return 7
	default:
		return -1
	}
}
func denseHigh(n denseWord) int {
	switch n {
	case 2147483648:
		return 1
	case 2147483649:
		return 2
	case 2147483650:
		return 3
	case 2147483652:
		return 5
	case 2147483653:
		return 6
	case 2147483654:
		return 7
	default:
		return -1
	}
}
func denseWrap(n denseWord) int {
	switch n {
	case 4294967293:
		return 1
	case 4294967294:
		return 2
	case 4294967295:
		return 3
	case 1:
		return 5
	case 2:
		return 6
	case 3:
		return 7
	}
	return -1
}
func denseFall(n int32) int {
	result := 0
	switch n {
	case 0:
		result++
		fallthrough
	default:
		result += 2
	case 1, 2:
		result += 4
		fallthrough
	case 3:
		result += 8
	case 4:
		result += 16
	case 5:
		result += 32
	case 6:
		result += 64
	}
	return result
}
func denseOrdered(n int32) int {
	switch denseEffect(n) {
	case 0:
		return 1
	case 1:
		return 2
	case 2:
		return 3
	case denseEffect(3):
		return 4
	case 4:
		return 5
	case 5:
		return 6
	case 6:
		return 7
	default:
		return -1
	}
}
func denseLoop(n int) int {
	sum := 0
outer:
	for i := 0; i < n; i++ {
		switch int32(i % 9) {
		case 0:
			continue outer
		case 1:
			sum += 1
		case 2:
			sum += 2
		case 3:
			sum += 3
		case 4:
			sum += 4
		case 5:
			sum += 5
		case 6:
			sum += 6
		default:
			sum += 9
		}
	}
	return sum
}

type denseRecord struct {
	Kind int
	A    int
	B    int
	Imm  uint64
}

func denseGrouped(o denseRecord, scratch int) bool {
	x, y, r, t, u, v := scratch, scratch+1, scratch+2, scratch+3, scratch+4, scratch+5
	width := o.Imm & 127
	switch o.Kind {
	case 42:
		denseEffects = x
	case 43:
		denseEffects = y
	case 44, 33, 34:
		denseEffects = r
	case 29, 30:
		denseEffects = t
	case 32, 31:
		denseEffects = u
	case 27, 28:
		denseEffects = v
	case 38, 39, 40, 41:
		denseEffects = int(width)
	default:
		return false
	}
	return true
}

func appMain(args []string) int {
	for k := 20; k <= 50; k++ {
		denseEffects = -1
		hit := denseGrouped(denseRecord{Kind: k, Imm: 64}, 8)
		want := k >= 27 && k <= 34 || k >= 38 && k <= 44
		if hit != want {
			print("FAIL grouped hit\n")
			return k
		}
		if k >= 38 && k <= 41 && denseEffects != 64 {
			print("FAIL grouped width\n")
			return 1
		}
	}

	for n := -5; n <= 12; n++ {
		want := -1
		if n == -2 || n == 0 {
			want = 100
		}
		if n == -1 {
			want = 99
		}
		if n == 2 || n >= 4 && n <= 7 {
			want = 100 + n
		}
		if denseSigned(int32(n)) != want {
			print("FAIL dense signed\n")
			return 1
		}
	}
	for d := -2; d <= 10; d++ {
		want := -1
		if d >= 0 && d <= 6 && d != 3 {
			want = d + 1
		}
		if denseLow(int32(-2147483648)+int32(d)) != want {
			print("FAIL low\n")
			return 1
		}
		if denseHigh(denseWord(2147483648)+denseWord(d)) != want {
			print("FAIL high\n")
			return 1
		}
		if denseWrap(denseWord(4294967293)+denseWord(d)) != want {
			print("FAIL wrap\n")
			return 1
		}
	}
	if denseFall(0) != 3 || denseFall(1) != 12 || denseFall(2) != 12 || denseFall(3) != 8 || denseFall(4) != 16 || denseFall(5) != 32 || denseFall(6) != 64 || denseFall(7) != 2 {
		print("FAIL dense fall\n")
		return 1
	}
	for n := -1; n <= 8; n++ {
		denseEffects = 0
		want := -1
		if n >= 0 && n <= 6 {
			want = n + 1
		}
		count := 2
		if n >= 0 && n <= 2 {
			count = 1
		}
		if denseOrdered(int32(n)) != want || denseEffects != count {
			print("FAIL dense effects\n")
			return 1
		}
	}
	denseEffects = 0
	if denseSigned(denseEffect(5)) != 105 || denseEffects != 1 {
		print("FAIL dense once\n")
		return 1
	}
	if denseLoop(90) != 390 {
		print("FAIL dense loop\n")
		return 1
	}
	print("PASS\n")
	return 0
}
