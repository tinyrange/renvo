package main

func appendU32(out []byte, v int) []byte {
	for i := 0; i < 5; i++ {
		b := byte(v & 0x7f)
		v = v >> 7
		if i == 4 {
			b = b & 0x0f
			out = append(out, b)
			return out
		}
		if v == 0 {
			out = append(out, b)
			return out
		}
		b = b | 0x80
		out = append(out, b)
	}
	return out
}

func appMain(args []string, env []string) int {
	out := appendU32(nil, -1)
	if len(out) != 5 {
		print("bad len\n")
		return 1
	}
	if out[0] != 0xff || out[1] != 0xff || out[2] != 0xff || out[3] != 0xff || out[4] != 0x0f {
		print("bad bytes\n")
		return 1
	}
	print("PASS\n")
	return 0
}
