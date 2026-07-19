package main

func renvoSliceTwoBoundEmit(data []byte, start int, end int) {
	write(1, data[start:end], -1)
}

func appMain(args []string) int {
	var data []byte
	data = append(data, 'x')
	data = append(data, 'P')
	data = append(data, 'A')
	data = append(data, 'S')
	data = append(data, 'S')
	data = append(data, '\n')
	data = append(data, 'x')
	renvoSliceTwoBoundEmit(data, 1, 6)
	return 0
}
