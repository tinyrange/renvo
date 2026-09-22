package main

func localPointer(x int) *int { v := x; return &v }
func paramPointer(x int) *int { return &x }
func appMain(args []string) int {
	p := localPointer(10)
	q := localPointer(20)
	if p == q || *p != 10 || *q != 20 {
		return 1
	}
	a := paramPointer(30)
	b := paramPointer(40)
	if a == b || *a != 30 || *b != 40 {
		return 2
	}
	print("PASS\n")
	return 0
}
