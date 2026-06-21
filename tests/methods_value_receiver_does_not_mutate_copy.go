package main

type rtgMD36Counter struct {
	value int
}

func (c rtgMD36Counter) AddCopy(n int) int {
	c.value += n
	return c.value
}

func appMain(args []string) int {
	c := rtgMD36Counter{value: 4}
	got := c.AddCopy(6)
	if got != 10 {
		print("methods_value_receiver_does_not_mutate_copy return failed\n")
		return 1
	}
	if c.value != 4 {
		print("methods_value_receiver_does_not_mutate_copy receiver failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
