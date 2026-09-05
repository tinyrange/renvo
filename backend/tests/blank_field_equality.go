package main

type BlankFields struct {
	_ int
	X int
	_ string
}

func appMain(args []string) int {
	a := BlankFields{1, 2, "a"}
	b := BlankFields{9, 2, "b"}
	if a != b {
		return 1
	}
	var x interface{} = a
	var y interface{} = b
	if x != y {
		return 2
	}
	b.X = 3
	if a == b {
		return 3
	}
	print("PASS\n")
	return 0
}
