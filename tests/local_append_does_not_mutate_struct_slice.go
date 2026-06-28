package main

type imageBuf struct {
	code []byte
}

func buildImage(buf *imageBuf) []byte {
	var out []byte
	out = append(out, 0x7f)
	out = append(out, 'E')
	out = append(out, 'L')
	out = append(out, 'F')
	for i := 0; i < len(buf.code); i++ {
		out = append(out, buf.code[i])
	}
	return out
}

func appMain() int {
	var buf imageBuf
	buf.code = make([]byte, 0, 64)
	buf.code = append(buf.code, 1)
	buf.code = append(buf.code, 2)
	buf.code = append(buf.code, 3)
	out := buildImage(&buf)
	if len(buf.code) == 3 && buf.code[0] == 1 && buf.code[1] == 2 && buf.code[2] == 3 && len(out) == 7 && out[0] == 0x7f && out[4] == 1 {
		print("PASS\n")
	}
	return 0
}
