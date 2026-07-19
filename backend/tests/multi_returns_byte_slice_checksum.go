package main

func renvo1015Build() ([]byte, int) {
	var data []byte
	data = append(data, 'A')
	data = append(data, 'C')
	return data, int(data[0]) + int(data[1])
}

func appMain(args []string) int {
	data, sum := renvo1015Build()
	if len(data) != 2 || sum != 132 {
		print("RENVO-1015 byte slice checksum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
