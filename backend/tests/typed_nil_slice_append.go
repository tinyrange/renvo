package main

func copyWords(words []uint32) []uint32 { return append([]uint32(nil), words...) }
func appMain(args []string) int {
	words := copyWords([]uint32{7, 9})
	if len(words) != 2 || words[0] != 7 || words[1] != 9 {
		panic("slice clone")
	}
	if copyWords(nil) != nil {
		panic("nil clone")
	}
	print("PASS\n")
	return 0
}
