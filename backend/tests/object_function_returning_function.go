package main

func returnedFunctionTarget() int {
	return 42
}

func returnFunction(which int) func() int {
	if which != 0 {
		return returnedFunctionTarget
	}
	return nil
}

func appMain(args []string) int {
	function := returnFunction(1)
	if function == nil || function() != 42 {
		return 1
	}
	print("PASS\n")
	return 0
}
