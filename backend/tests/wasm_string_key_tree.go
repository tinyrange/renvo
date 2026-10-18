package main

func dispatchKeyString(s string) int {
	switch s {
	case "":
		return 1
	case "a":
		return 2
	case "f":
		return 3
	case "m":
		return 4
	case "z":
		return 5
	case "\x80":
		return 6
	case "\xff":
		return 7
	case "ab":
		return 8
	case "ac":
		return 9
	case "mb":
		return 10
	case "za":
		return 11
	case "abc":
		return 12
	case "acc":
		return 13
	case "mbc":
		return 14
	case "zab":
		return 15
	case "aaaa":
		return 16
	case "bbbb":
		return 17
	case "cccc":
		return 18
	case "abc\x7f":
		return 19
	case "abc\x80":
		return 20
	case "abc\xfe":
		return 21
	case "abc\xff":
		return 22
	case "\x00\x00\x00\x80":
		return 23
	case "\xff\xff\xff\xff":
		return 24
	case "readcr":
		return 25
	case "readdr":
		return 26
	case "readsg":
		return 27
	case "alphaa":
		return 28
	case "zuluuu":
		return 29
	case "readtail":
		return 30
	case "readnail":
		return 31
	case "abc\x80tail":
		return 32
	case "readabcdefghijkl":
		return 33
	case "readabcdefghijkm":
		return 34
	case "abc\xffabcdefghijkl":
		return 35
	case "multi-aa", "multi-bb", "secondcc":
		return 99
	default:
		return -1
	}
}
func keyFallthrough(s string) int {
	n := 0
	switch s {
	case "read00":
		n += 1
		fallthrough
	default:
		n += 2
	case "read01", "abcd00":
		n += 4
		fallthrough
	case "read02":
		n += 8
	case "zzzz00":
		n += 16
	case "mmmm00":
		n += 32
	case "aaaa00":
		n += 64
	}
	return n
}
func appMain(args []string) int {
	values := []string{"", "a", "f", "m", "z", "\x80", "\xff", "ab", "ac", "mb", "za", "abc", "acc", "mbc", "zab", "aaaa", "bbbb", "cccc", "abc\x7f", "abc\x80", "abc\xfe", "abc\xff", "\x00\x00\x00\x80", "\xff\xff\xff\xff", "readcr", "readdr", "readsg", "alphaa", "zuluuu", "readtail", "readnail", "abc\x80tail", "readabcdefghijkl", "readabcdefghijkm", "abc\xffabcdefghijkl"}
	for i := 0; i < len(values); i++ {
		if dispatchKeyString(values[i]) != i+1 {
			print("FAIL key hit ", i, " got ", dispatchKeyString(values[i]), "\n")
			return 1
		}
		// Exercise every prefix and tail mismatch, including keys missing
		// between tree nodes and signed keys above and below each pivot.
		for pos := 0; pos < len(values[i]); pos++ {
			bytes := []byte(values[i])
			bytes[pos] ^= 0x40
			changed := string(bytes)
			want := -1
			for j := 0; j < len(values); j++ {
				if changed == values[j] {
					want = j + 1
					break
				}
			}
			if dispatchKeyString(changed) != want {
				print("FAIL key miss\n")
				return 1
			}
		}
		// Deliberately unaligned input with unrelated bytes on both sides.
		padded := []byte("!" + values[i] + "?")
		if dispatchKeyString(string(padded[1:len(padded)-1])) != i+1 {
			print("FAIL unaligned key\n")
			return 1
		}
	}
	if dispatchKeyString("multi-aa") != 99 || dispatchKeyString("multi-bb") != 99 || dispatchKeyString("secondcc") != 99 || dispatchKeyString("multi-cc") != -1 {
		print("FAIL key list\n")
		return 1
	}
	if keyFallthrough("read00") != 3 || keyFallthrough("read01") != 12 || keyFallthrough("abcd00") != 12 || keyFallthrough("read02") != 8 || keyFallthrough("read03") != 2 || keyFallthrough("zzzz00") != 16 || keyFallthrough("aaaa00") != 64 {
		print("FAIL key fallthrough\n")
		return 1
	}
	print("PASS\n")
	return 0
}
