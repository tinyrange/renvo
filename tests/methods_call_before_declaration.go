package main

type rtgMD39Box struct {
	a int
	b int
}

func appMain(args []string) int {
	box := rtgMD39Box{a: 10, b: 5}
	if box.Diff() != 5 {
		print("methods_call_before_declaration failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func (b rtgMD39Box) Diff() int {
	return b.a - b.b
}
