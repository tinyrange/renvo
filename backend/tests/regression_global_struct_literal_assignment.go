package main

type renvoGlobalStructLiteralPair struct {
	left  int
	right int
}

var renvoGlobalStructLiteralValue renvoGlobalStructLiteralPair

func appMain(args []string) int {
	renvoGlobalStructLiteralValue = renvoGlobalStructLiteralPair{left: 4, right: 8}
	if renvoGlobalStructLiteralValue.left+renvoGlobalStructLiteralValue.right != 12 {
		print("global struct literal keyed assignment failed\n")
		return 1
	}
	renvoGlobalStructLiteralValue = renvoGlobalStructLiteralPair{9, 3}
	if renvoGlobalStructLiteralValue.left+renvoGlobalStructLiteralValue.right != 12 {
		print("global struct literal positional assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
