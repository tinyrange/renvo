package main

const highCase = "\x80\xff\x00\xfe" + "tail"
const namedCase = "abc" + string("def")

var effects int

func selectString(s string) int {
	switch s {
	case "":
		return 0
	case "a":
		return 1
	case "ab":
		return 2
	case "abc":
		return 3
	case "abcd":
		return 4
	case "abcde":
		return 5
	case namedCase:
		return 6
	case "abcdefg":
		return 7
	case "abcdefgh":
		return 8
	case "abcdefghi":
		return 9
	case "abcdefghij":
		return 10
	case "abcdefghijk":
		return 11
	case "abcdefghijkl":
		return 12
	case "abcdefghijklm":
		return 13
	case "abcdefghijklmn":
		return 14
	case "abcdefghijklmno":
		return 15
	case "abcdefghijklmnop":
		return 16
	case highCase:
		return 24
	case "x", "yy", "zzzz":
		return 60
	default:
		return -1
	}
}
func effectString(s string) string { effects++; return s }
func orderedString(s string) int {
	switch effectString(s) {
	case effectString("no"):
		return 1
	case "abcd", effectString("abce"):
		return 2
	case highCase:
		return 3
	case "other1", "other2", "other3":
		return 5
	case "other4":
		return 6
	case "other5":
		return 7
	default:
		return 4
	}
}
func fallString(s string) int {
	n := 0
	switch s {
	case "abcd":
		n += 1
		fallthrough
	default:
		n += 2
	case "abcde":
		n += 4
		fallthrough
	case "abcdef":
		n += 8
	case "unseen1":
		n += 16
	case "unseen2":
		n += 32
	case "unseen3":
		n += 64
	}
	return n
}
func compareString(s string) bool {
	return s == "abcdefghijklmno" && "abcdefghijklmno" == s && !(s != "abcdefghijklmno")
}
func appMain(args []string) int {
	data := []byte("abcdefghijklmnopq")
	for n := 0; n <= 17; n++ {
		text := string(data[:n])
		want := n
		if n == 17 {
			want = -1
		}
		if selectString(text) != want {
			print("FAIL length\n")
			return 1
		}
		for offset := 0; offset < n; offset++ {
			data[offset] ^= 0x80
			if selectString(string(data[:n])) != -1 {
				print("FAIL differing byte\n")
				return 1
			}
			data[offset] ^= 0x80
		}
	}
	if selectString("x") != 60 || selectString("yy") != 60 || selectString("zzzz") != 60 {
		print("FAIL mixed case lists\n")
		return 1
	}
	// Unaligned data, high bytes, embedded NUL, empty/nil descriptor, and tails.
	bytes := []byte{9, 0x80, 0xff, 0, 0xfe, 't', 'a', 'i', 'l', 9}
	if selectString(string(bytes[1:9])) != 24 || selectString(string(bytes[1:8])) != -1 || selectString(string([]byte(nil))) != 0 {
		print("FAIL binary\n")
		return 1
	}
	if !compareString(string(data[:15])) || compareString(string(data[:14])) || compareString(string(data[:16])) {
		print("FAIL equality\n")
		return 1
	}
	effects = 0
	if orderedString(string(data[:4])) != 2 || effects != 2 {
		print("FAIL order early\n")
		return 1
	}
	effects = 0
	if orderedString("abce") != 2 || effects != 3 {
		print("FAIL order late\n")
		return 1
	}
	effects = 0
	if orderedString(string(bytes[1:9])) != 3 || effects != 3 {
		print("FAIL order constant\n")
		return 1
	}
	if fallString(string(data[:4])) != 3 || fallString(string(data[:5])) != 12 || fallString(string(data[:6])) != 8 || fallString("none") != 2 {
		print("FAIL fallthrough\n")
		return 1
	}
	effects = 0
	if effectString("abcd") == "abcde" || effects != 1 {
		print("FAIL operand once\n")
		return 1
	}
	print("PASS\n")
	return 0
}
