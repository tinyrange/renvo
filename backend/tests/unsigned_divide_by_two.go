package main
func appMain() int {
	maximum := ^uint(0)
	half := maximum / 2
	if half != maximum >> 1 { panic("unsigned local division") }
	print("PASS\n")
	return 0
}
