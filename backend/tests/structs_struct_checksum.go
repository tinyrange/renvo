package main

type Renvo0625Record struct {
	a int
	b byte
	c bool
}

func appMain(args []string) int {
	r := Renvo0625Record{a: 5, b: 'A', c: true}
	score := r.a + int(r.b)
	if r.c {
		score = score + 1
	}
	if score != 71 {
		print("RENVO-0625 struct checksum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
