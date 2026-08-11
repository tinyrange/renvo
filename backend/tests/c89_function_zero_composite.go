package main

type c89FunctionZeroComposite func(int) int

func appMain() int {
	var value c89FunctionZeroComposite = nil
	if value != nil {
		return 1
	}
	print("PASS\n")
	return 0
}
