package main

type sharedNameReceiver int

type sharedNameResult struct {
	left  int
	right int
}

func sharedNameValue() sharedNameResult {
	return sharedNameResult{left: 40, right: 2}
}

func (sharedNameReceiver) sharedNameValue() int {
	return -1
}

func appMain(args []string) int {
	value := sharedNameValue()
	if value.left+value.right != 42 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
