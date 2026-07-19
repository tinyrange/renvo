package main

type renvoStructReturnAfterGlobalSetResult struct {
	data []byte
	ok   bool
}

var renvoStructReturnAfterGlobalSetState int

func renvoStructReturnAfterGlobalSetMark(v int) {
	if renvoStructReturnAfterGlobalSetState == 0 {
		renvoStructReturnAfterGlobalSetState = v
	}
}

func renvoStructReturnAfterGlobalSetFail(v int) renvoStructReturnAfterGlobalSetResult {
	renvoStructReturnAfterGlobalSetMark(v)
	var result renvoStructReturnAfterGlobalSetResult
	return result
}

func appMain(args []string) int {
	result := renvoStructReturnAfterGlobalSetFail(7)
	if renvoStructReturnAfterGlobalSetState != 7 || result.ok || len(result.data) != 0 {
		print("struct return after global set failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
