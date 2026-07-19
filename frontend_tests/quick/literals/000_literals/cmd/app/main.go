package main

const hexValue = 0x3e

func append16(out []byte, v int) []byte {
	out = append(out, byte(v))
	out = append(out, byte(v>>8))
	return out
}

func main() {
	var out []byte
	out = append16(out, hexValue)
	if len(out) != 2 {
		print("FAIL\n")
		return
	}
	if out[0] != 0x3e {
		print("FAIL\n")
		return
	}
	if out[1] != 0 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
