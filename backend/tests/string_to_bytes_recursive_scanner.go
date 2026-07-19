package main

var renvo0597Want byte = 'k'

func renvo0597Find(bs []byte, i int) int {
	if i >= len(bs) {
		return -1
	}
	if bs[i] == renvo0597Want {
		return i
	}
	return renvo0597Find(bs, i+1)
}

func appMain(args []string) int {
	if renvo0597Find([]byte("stack"), 0) != 4 {
		print("RENVO-0597 recursive scanner failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
