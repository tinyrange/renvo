package main

var global int = 10

func appMain(args []string) int {
	x := 1
	{
		var x = x + 1
		if x != 2 {
			return 1
		}
	}
	{
		x := x + 2
		if x != 3 {
			return 2
		}
	}
	{
		var global = global + 1
		if global != 11 {
			return 3
		}
	}
	if x != 1 || global != 10 {
		return 4
	}
	print("PASS\n")
	return 0
}
