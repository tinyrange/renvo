package main

type Op struct {
	Kind int
	A, B int
	Imm  uint64
}
type Entry struct {
	key   Op
	value int
}

func equal(a, b Op) bool { return a == b }
func find(entries []Entry, key Op) int {
	for i := 0; i < len(entries); i++ {
		if entries[i].key == key {
			return i
		}
	}
	return -1
}
func appMain(args []string) int {
	a := Op{Kind: 0, Imm: 0xfe00000000000000}
	b := Op{Kind: 0, Imm: 0}
	if equal(a, b) || find([]Entry{{key: a, value: 6}}, b) != -1 {
		print("FAIL high-word equality\n")
		return 1
	}
	print("PASS\n")
	return 0
}
