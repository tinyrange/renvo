package main

type rtgUnaryNotStructResult struct {
	value int
}

func rtgUnaryNotStructCheck(i int) bool {
	return i == 1
}

func rtgUnaryNotStructMake() rtgUnaryNotStructResult {
	var result rtgUnaryNotStructResult
	i := 0
	i++
	if rtgUnaryNotStructCheck(i) {
	} else {
		return result
	}
	if !rtgUnaryNotStructCheck(i) {
		return result
	}
	result.value = 1
	return result
}

func appMain(args []string, env []string) int {
	result := rtgUnaryNotStructMake()
	if result.value == 1 {
		print("PASS\n")
	}
	return 0
}
