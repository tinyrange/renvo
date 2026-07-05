package main

func append16HexLiteral(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	return out
}

func appMain(args []string, env []string) int {
	var out []byte
	out = append16HexLiteral(out, 0x3e)
	if len(out) != 2 {
		print("FAIL\n")
		return 1
	}
	if out[0] != 0x3e {
		print("FAIL\n")
		return 1
	}
	if out[1] != 0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
