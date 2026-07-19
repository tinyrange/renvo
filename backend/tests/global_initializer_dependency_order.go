package main

var initDependencyTrace int

func initDependencyMark(value int) int {
	initDependencyTrace = initDependencyTrace*10 + value
	return value
}

var initDependencyFirst = initDependencyMark(1)
var initDependencyTotal = initDependencyLater + initDependencyMark(3)
var initDependencyLater = initDependencyMark(2)
var initDependencyViaFunction = initDependencyReadLast()
var initDependencyLast = 4

func initDependencyReadLast() int {
	return initDependencyLast
}

func appMain(args []string) int {
	if initDependencyFirst != 1 || initDependencyTotal != 5 || initDependencyLater != 2 || initDependencyViaFunction != 4 || initDependencyTrace != 123 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
