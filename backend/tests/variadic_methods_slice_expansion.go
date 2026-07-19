package main

type renvoVM49Box struct {
	total int
}

func (box *renvoVM49Box) Add(values ...int) {
	i := 0
	for i < len(values) {
		box.total += values[i]
		i += 1
	}
}

func appMain(args []string) int {
	values := []int{4, 6}
	box := renvoVM49Box{total: 1}
	box.Add(values...)
	if box.total != 11 {
		print("variadic_methods_slice_expansion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
