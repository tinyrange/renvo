package main

func isBool(x interface{}) bool { _, ok := x.(bool); return ok }
func appMain(args []string) int {
	x := 1
	var p *int
	b := x == 1
	if !isBool(x == 1) || !isBool(x < 2) || !isBool(p == nil) || !isBool(!b) || !isBool(b && x < 2) || !isBool(b) {
		return 1
	}
	print("PASS\n")
	return 0
}
