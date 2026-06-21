package main

type rtgCP25Byte byte

func appMain(args []string) int {
	source := []rtgCP25Byte{rtgCP25Byte('A'), rtgCP25Byte('B'), rtgCP25Byte('C')}
	dest := make([]rtgCP25Byte, 3)
	n := copy(dest, source)
	if n != 3 {
		print("copy_builtin_named_byte_slice count failed\n")
		return 1
	}
	if int(dest[0])+int(dest[2]) != int('A')+int('C') {
		print("copy_builtin_named_byte_slice value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
