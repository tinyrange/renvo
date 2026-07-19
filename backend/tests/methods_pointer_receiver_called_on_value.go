package main

type renvoMD37Counter struct {
	value int
}

func (c *renvoMD37Counter) Bump() {
	c.value = c.value + 1
}

func appMain(args []string) int {
	c := renvoMD37Counter{value: 8}
	c.Bump()
	if c.value != 9 {
		print("methods_pointer_receiver_called_on_value failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
