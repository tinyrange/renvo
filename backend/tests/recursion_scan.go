package main

func renvo0506Find(s string, idx int, want byte) int {
	if idx >= len(s) {
		return -1
	}
	if s[idx] == want {
		return idx
	}
	return renvo0506Find(s, idx+1, want)
}

func appMain(args []string) int {
	if renvo0506Find("parcel", 0, 'c') != 3 {
		print("RENVO-0506 scan failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
