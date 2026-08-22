package main

type dereferencedStructArrayHeader struct {
	ident [4]byte
}

func dereferencedStructArrayInvalid(header *dereferencedStructArrayHeader) bool {
	return int32((*header).ident[0]) != 127 || int32((*header).ident[1]) != 69 || int32((*header).ident[2]) != 76 || int32((*header).ident[3]) != 70
}

func appMain(args []string) int {
	value := dereferencedStructArrayHeader{ident: [4]byte{127, 69, 76, 70}}
	if dereferencedStructArrayInvalid(&value) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
