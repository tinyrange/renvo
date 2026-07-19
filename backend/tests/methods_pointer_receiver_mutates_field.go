package main

type renvoMD35Counter struct {
	value int
}

func (c *renvoMD35Counter) Add(n int) {
	c.value += n
}

func appMain(args []string) int {
	c := renvoMD35Counter{value: 6}
	c.Add(5)
	if c.value != 11 {
		print("methods_pointer_receiver_mutates_field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
