package main

type rtgMD35Counter struct {
	value int
}

func (c *rtgMD35Counter) Add(n int) {
	c.value += n
}

func appMain(args []string) int {
	c := rtgMD35Counter{value: 6}
	c.Add(5)
	if c.value != 11 {
		print("methods_pointer_receiver_mutates_field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
