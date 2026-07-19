package main

var compilerDefaultTarget int = 31
var compilerFixedTarget int = 32
var currentTarget int = 33
var targetArch int = 34
var targetOS int = 35
var privateNameCalls int

func trustNonNil(value int) {
	privateNameCalls += value
}

func privateNamesMatch() bool {
	return compilerDefaultTarget == 31 &&
		compilerFixedTarget == 32 &&
		currentTarget == 33 &&
		targetArch == 34 &&
		targetOS == 35
}

func appMain(args []string) int {
	trustNonNil(7)
	if !privateNamesMatch() || privateNameCalls != 7 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
