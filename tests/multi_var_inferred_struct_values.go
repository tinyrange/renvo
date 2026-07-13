package main

type multiVarValue struct {
	n int
}

func appMain(args []string) int {
	var first, second = multiVarValue{n: 1}, multiVarValue{n: 2}
	if first.n != 1 || second.n != 2 {
		print("multi_var_inferred_struct_values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
