package main

type globalInitializerBox struct {
	value int
	mark  byte
}

var globalInitializerCalls int

func globalInitializerValue(value int) int {
	globalInitializerCalls++
	return value
}

var globalInitializedBox = globalInitializerBox{
	value: globalInitializerValue(41),
	mark:  byte(globalInitializerValue(66)),
}

func appMain() int {
	if globalInitializerCalls != 2 || globalInitializedBox.value != 41 || globalInitializedBox.mark != 'B' {
		print("global composite initializer call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
