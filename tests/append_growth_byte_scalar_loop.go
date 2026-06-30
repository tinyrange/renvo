package main

func appendBytes(out []byte, values []byte) []byte {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func appMain(args []string) int {
	capacity := 1200
	out := make([]byte, 0, capacity)
	values := []byte("abcdefghijklmnopqrstuvwxyz")
	i := 0
	for i < 50 {
		out = appendBytes(out, values)
		i++
	}
	if len(out) != 1300 {
		print("FAIL\n")
		return 1
	}
	if out[0] != 'a' || out[25] != 'z' || out[26] != 'a' || out[1299] != 'z' {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
