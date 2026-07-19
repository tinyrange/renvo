package main

func renvoVF45Checksum(values ...byte) int {
	total := 0
	i := 0
	for i < len(values) {
		total += int(values[i])
		i += 1
	}
	return total
}

func appMain(args []string) int {
	if renvoVF45Checksum('A', byte(2), 'C') != int('A')+2+int('C') {
		print("variadic_functions_byte_checksum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
