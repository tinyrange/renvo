package main

type renvoVM48Box struct {
	total int
}

func (box *renvoVM48Box) Add(values ...int) {
	i := 0
	for i < len(values) {
		box.total += values[i]
		i += 1
	}
}

func appMain(args []string) int {
	box := renvoVM48Box{}
	box.Add(2, 3, 5)
	if box.total != 10 {
		print("variadic_methods_pointer_receiver_accumulates failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
