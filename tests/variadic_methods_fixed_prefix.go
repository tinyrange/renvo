package main

type rtgVM50Writer struct {
	total int
}

func (writer *rtgVM50Writer) Add(base byte, values ...byte) {
	writer.total = int(base)
	i := 0
	for i < len(values) {
		writer.total += int(values[i])
		i += 1
	}
}

func appMain(args []string) int {
	writer := rtgVM50Writer{}
	writer.Add('a', 'b', 'c')
	if writer.total != int('a')+int('b')+int('c') {
		print("variadic_methods_fixed_prefix failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
