package main

type boundWord uint32
type boundBox struct {
	values [256]int
	nested [2][256]int
}

var boundCalls int
var boundGlobal boundBox

func boundIndex(v int) int { boundCalls++; return v }
func boundBase() *boundBox { boundCalls++; return &boundGlobal }
func boundArray() [256]int { boundCalls++; var a [256]int; a[255] = 19; return a }
func boundPanic(kind int, i uint64) (caught bool) {
	defer func() { caught = recover() != nil }()
	switch kind {
	case 0:
		a := []int{1, 2}
		_ = a[boundIndex(255)&255]
	case 1:
		var a [255]int
		_ = a[boundIndex(255)&255]
	case 2:
		var a [0]int
		_ = a[boundIndex(0)&0]
	case 3:
		var a *[256]int
		_ = a[boundIndex(255)&255]
	case 4:
		var a [256]int
		_ = a[i&((uint64(1)<<32)|255)]
	case 5:
		var a [256]int
		_ = a[int32(boundIndex(-1))&int32(-1)]
	case 6:
		var a [256]int
		_ = a[boundIndex(-1)&^255]
	case 7:
		s := []int{1, 2}
		for j := 0; j < len(s); j++ {
			if j == 1 {
				s = s[:0]
			}
			_ = s[j]
		}
	case 8:
		s := []int{1, 2}
		for j := 0; j < len(s); j++ {
			if j == 0 {
				j = 4
			}
			_ = s[j]
		}
	case 9:
		s := []int{1, 2}
		p := &s
		for j := 0; j < len(s); j++ {
			*p = (*p)[:0]
			_ = s[j]
		}
	case 10:
		s := []int{1, 2}
		for j := 0; j < len(s); j++ {
			p := &j
			*p = 3
			_ = s[j]
		}
	case 11:
		s := []int{1, 2}
		shrink := func() { s = s[:0] }
		for j := 0; j < len(s); j++ {
			shrink()
			_ = s[j]
		}
	case 12:
		s := []int{1, 2}
		for j := 0; j < len(s); j++ {
			bad := func() { j = 3 }
			bad()
			_ = s[j]
		}
	case 13:
		s := []int{1, 2}
		len := func([]int) int { return 4 }
		for j := 0; j < len(s); j++ {
			_ = s[j]
		}
	}
	return
}
func appMain() int {
	var b boundBox
	for _, i := range []int{-2147483647 - 1, -513, -257, -256, -255, -128, -1, 0, 1, 127, 128, 254, 255, 256, 257, 513, 2147483647} {
		j := i & 255
		b.values[i&255] = j + 1
		b.values[255&i] += 2
		p := &b.values[uint32(i)&255]
		*p += 3
		b.nested[1][int32(i)&255] = *p
		if b.values[boundWord(i)&255] != j+6 || b.nested[1][byte(i)] != j+6 {
			return 1
		}
	}
	var one [1]int
	one[boundIndex(-1)&0] = 7
	var almost [255]int
	almost[boundIndex(-1)&127] = 8
	var above [257]int
	above[byte(255)] = 9
	if one[0] != 7 || almost[127] != 8 || above[255] != 9 {
		return 2
	}
	var medium [65536]byte
	medium[uint16(65535)] = 11
	if medium[65535] != 11 {
		return 3
	}
	boundCalls = 0
	boundBase().values[boundIndex(-1)&255] = 13
	if boundCalls != 2 || boundGlobal.values[255] != 13 {
		return 4
	}
	if boundArray()[boundIndex(-1)&255] != 19 || boundCalls != 4 {
		return 5
	}
	s := []int{2, 3, 4}
	total := 0
	for j := 0; j < len(s); j++ {
		total += s[j]
	}
	if total != 9 {
		return 6
	}
	// Nested facts must not escape their lexical loop, including a goto exit.
	for j := 0; j < len(s); j++ {
		for k := 0; k < len(s); k++ {
			total += s[k]
		}
		if j == 1 {
			goto done
		}
	}
done:
	if total != 27 {
		return 7
	}
	for kind := 0; kind < 14; kind++ {
		if !boundPanic(kind, uint64(1)<<32) {
			return 20 + kind
		}
	}
	print("PASS\n")
	return 0
}
