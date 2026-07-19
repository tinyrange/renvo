package main

type renvoMD38Pair struct {
	left  int
	right int
}

func (p renvoMD38Pair) Product() int {
	return p.left * p.right
}

func appMain(args []string) int {
	p := renvoMD38Pair{left: 3, right: 4}
	ptr := &p
	if ptr.Product() != 12 {
		print("methods_value_method_called_on_pointer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
