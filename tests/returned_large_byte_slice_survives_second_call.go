package main

func buildLarge(tag byte) []byte {
	out := make([]byte, 0, 70000)
	for i := 0; i < 70000; i++ {
		out = append(out, byte(i%251))
	}
	out[0] = tag
	out[69999] = tag
	return out
}

func appMain(args []string, env []string) int {
	first := buildLarge('A')
	second := buildLarge('B')
	if len(first) != 70000 || len(second) != 70000 {
		print("FAIL len\n")
		return 1
	}
	if first[0] != 'A' || first[69999] != 'A' {
		print("FAIL first\n")
		return 1
	}
	if second[0] != 'B' || second[69999] != 'B' {
		print("FAIL second\n")
		return 1
	}
	print("PASS\n")
	return 0
}
