package main

type renvoUnaryNotLargeResult struct {
	src   []byte
	toks  []int
	decls []int
	funcs []int
	ok    bool
}

func renvoUnaryNotLargeCheck(i int) bool {
	return i == 1
}

func renvoUnaryNotLargeMake() renvoUnaryNotLargeResult {
	var result renvoUnaryNotLargeResult
	i := 0
	i++
	if renvoUnaryNotLargeCheck(i) {
	} else {
		return result
	}
	if !renvoUnaryNotLargeCheck(i) {
		return result
	}
	result.ok = true
	return result
}

func appMain(args []string, env []string) int {
	result := renvoUnaryNotLargeMake()
	if result.ok {
		print("PASS\n")
	}
	return 0
}
