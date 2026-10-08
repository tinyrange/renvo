package main

type CPU struct{ State [2]uint64 }

func registers(c *CPU) []uint64 { return c.State[:] }
func appMain() int {
	c := CPU{}
	c.State[1] = 4096
	s := registers(&c)
	s[0] = 7
	s[1] = 4100
	if c.State[0] != 7 || c.State[1] != 4100 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
