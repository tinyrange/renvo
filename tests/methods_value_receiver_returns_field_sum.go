package main

type rtgMD34Point struct {
	x int
	y int
}

func (p rtgMD34Point) Sum() int {
	return p.x + p.y
}

func appMain(args []string) int {
	p := rtgMD34Point{x: 3, y: 4}
	if p.Sum() != 7 {
		print("methods_value_receiver_returns_field_sum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
