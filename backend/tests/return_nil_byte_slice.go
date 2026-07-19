package main

func renvoReturnNilByteSlice() []byte {
	return nil
}

func appMain() int {
	b := renvoReturnNilByteSlice()
	if len(b) != 0 {
		print("return_nil_byte_slice length failed\n")
		return 1
	}
	b = append(b, 'P')
	if len(b) != 1 {
		print("return_nil_byte_slice append length failed\n")
		return 1
	}
	if b[0] != 'P' {
		print("return_nil_byte_slice append value failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
