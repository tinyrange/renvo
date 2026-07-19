package main

func appMain(args []string) int {
	data := []byte{'r', 't', 'g'}
	if len(data) != 3 {
		print("slice_literals_byte_values length failed\n")
		return 1
	}
	if int(data[0])+int(data[1])+int(data[2]) != int('r')+int('t')+int('g') {
		print("slice_literals_byte_values checksum failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
