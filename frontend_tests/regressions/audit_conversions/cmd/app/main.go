package main

func shortArray() (ok bool) {
	defer func() { ok = recover() != nil }()
	s := []int{1}
	_ = [2]int(s)
	return
}

func main() {
	for _, r := range []rune{65, 955, 128578} {
		if len(string(r)) == 0 {
			panic("range rune conversion")
		}
	}
	b := byte(65)
	r := rune(0x1f600)
	i := -1
	u := uint64(0xffffffffffffffff)
	if string(b) != "A" || string(r) != "😀" || string(i) != "�" || string(u) != "�" {
		panic("integer strings")
	}
	const letter = string(0x754c)
	if letter != "界" {
		panic("constant string")
	}
	dest := make([]byte, 3)
	if copy(dest, "hello") != 3 || string(dest) != "hel" {
		panic("copy string")
	}
	s := []int{1, 2, 3}
	a := [3]int(s)
	a[0] = 42
	if s[0] != 1 || a[0] != 42 || !shortArray() {
		panic("array value")
	}
	if ^uint8(0) != 255 {
		panic("typed complement")
	}
	println("PASS")
}
