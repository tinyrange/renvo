package main

const namedConstantCallFlag = 1 << 1

func namedConstantCallConsume(value int64) {}

func appMain(args []string) int {
	namedConstantCallConsume(namedConstantCallFlag)
	print("PASS\n")
	return 0
}
