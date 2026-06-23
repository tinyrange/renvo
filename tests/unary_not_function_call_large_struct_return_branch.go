package main

type rtgUnaryNotLargeResult struct {
	src   []byte
	toks  []int
	decls []int
	funcs []int
	ok    bool
}

func rtgUnaryNotLargeCheck(i int) bool {
	return i == 1
}

func rtgUnaryNotLargeMake() rtgUnaryNotLargeResult {
	var result rtgUnaryNotLargeResult
	i := 0
	i++
	if rtgUnaryNotLargeCheck(i) {
	} else {
		return result
	}
	if !rtgUnaryNotLargeCheck(i) {
		return result
	}
	result.ok = true
	return result
}

func appMain(args []string, env []string) int {
	result := rtgUnaryNotLargeMake()
	if result.ok {
		print("PASS\n")
	}
	return 0
}
