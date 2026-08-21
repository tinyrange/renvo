package main

const UTFMax = 4

type arrayWriter struct{}

func (writer *arrayWriter) writeRune(r rune) (int, error) {
	if r < 0x80 {
		return 1, nil
	}
	var data [UTFMax]byte
	if r < 0 || r > 0x10ffff || (r >= 0xd800 && r <= 0xdfff) {
		r = 0xfffd
	}
	value := uint32(r)
	if r < 0x800 {
		data[0] = byte(0xc0 + value/64)
		data[1] = byte(0x80 + value%64)
	} else if r < 0x10000 {
		data[0] = byte(0xe0 + value/4096)
		data[1] = byte(0x80 + value/64&0x3f)
		data[2] = byte(0x80 + value%64)
	} else {
		data[0] = byte(0xf0 + value/262144)
		data[1] = byte(0x80 + value/4096&0x3f)
		data[2] = byte(0x80 + value/64&0x3f)
		data[3] = byte(0x80 + value%64)
	}
	if data[0] == 0xe2 && data[1] == 0x82 && data[2] == 0xac {
		return 3, nil
	}
	return 0, nil
}

func appMain() int {
	writer := &arrayWriter{}
	if n, err := writer.writeRune('€'); n == 3 && err == nil {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
