package main

type sharedNamed int32
type sharedOther int32
type sharedRecord struct {
	Label string
	Words [3]uint64
	_     byte
}

var sharedEffects int

func sharedEqual(a, b any) bool   { return a == b }
func sharedUnequal(a, b any) bool { return a != b }
func sharedChoose(v any) int {
	switch v {
	case "hit":
		return 1
	case uint64(0xfedcba9876543210):
		return 2
	case sharedNamed(7):
		return 3
	case nil:
		return 4
	default:
		return 5
	}
}
func sharedEffect(v any) any { sharedEffects++; return v }

type lateWord uint64
type lateRecord struct {
	Name string
	Key  lateWord
}

func sharedLateTypes() bool {
	a := lateRecord{Name: "late", Key: lateWord(0x8000000012345678)}
	b := a
	return sharedEqual(a, b) && !sharedUnequal(a, b)
}
func appMain(args []string) int {
	var absent any
	var p *int
	values := []any{nil, p, 7, sharedNamed(7), sharedOther(7), uint64(0xfedcba9876543210), "dynamic", true, sharedRecord{Label: "text", Words: [3]uint64{1, 0x8000000000000000, 3}}, [3]int16{-1, 0, 32767}}
	for i := 0; i < len(values); i++ {
		if !sharedEqual(values[i], values[i]) || sharedUnequal(values[i], values[i]) {
			print("FAIL shared self\n")
			return 1
		}
		for j := 0; j < len(values); j++ {
			if sharedEqual(values[i], values[j]) != (i == j) || sharedUnequal(values[i], values[j]) != (i != j) {
				print("FAIL shared cross\n")
				return 1
			}
		}
	}
	if !sharedEqual(absent, nil) || sharedEqual(p, nil) || sharedEqual(sharedNamed(7), sharedOther(7)) {
		print("FAIL shared types\n")
		return 1
	}
	if sharedChoose("hit") != 1 || sharedChoose(uint64(0xfedcba9876543210)) != 2 || sharedChoose(sharedNamed(7)) != 3 || sharedChoose(nil) != 4 || sharedChoose(7) != 5 {
		print("FAIL shared switch\n")
		return 1
	}
	a := sharedRecord{Label: "text", Words: [3]uint64{1, 2, 3}}
	b := a
	b.Words[2] = 4
	if sharedEqual(a, b) {
		print("FAIL shared aggregate\n")
		return 1
	}
	zero := float64(0)
	nan := zero / zero
	negativeZero := -zero
	if sharedEqual(nan, nan) || !sharedUnequal(nan, nan) || !sharedEqual(zero, negativeZero) {
		print("FAIL shared floats\n")
		return 1
	}
	f := float32(0)
	nf := f / f
	if sharedEqual(nf, nf) || !sharedEqual(f, -f) {
		print("FAIL shared float32\n")
		return 1
	}
	if !sharedLateTypes() {
		print("FAIL shared late\n")
		return 1
	}
	sharedEffects = 0
	if !sharedEqual(sharedEffect("ordered"), sharedEffect("ordered")) || sharedEffects != 2 {
		print("FAIL shared effects\n")
		return 1
	}
	print("PASS\n")
	return 0
}
