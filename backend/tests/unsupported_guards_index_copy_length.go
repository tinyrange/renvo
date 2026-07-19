package main

func appMain(args []string) int {
	source := "abcdef"
	var copy []byte
	i := 1
	total := 0
	for i < 5 {
		copy = append(copy, source[i])
		total += int(copy[len(copy)-1])
		i += 1
	}
	if len(copy) != 4 {
		print("RENVO-0844 index copy length failed\n")
		return 1
	}
	if total != int('b')+int('c')+int('d')+int('e') {
		print("RENVO-0844 index copy sum failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
