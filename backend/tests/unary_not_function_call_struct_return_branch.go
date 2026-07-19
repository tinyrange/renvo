package main

type renvoUnaryNotStructResult struct {
	value int
}

func renvoUnaryNotStructCheck(i int) bool {
	return i == 1
}

func renvoUnaryNotStructMake() renvoUnaryNotStructResult {
	var result renvoUnaryNotStructResult
	i := 0
	i++
	if renvoUnaryNotStructCheck(i) {
	} else {
		return result
	}
	if !renvoUnaryNotStructCheck(i) {
		return result
	}
	result.value = 1
	return result
}

func appMain(args []string, env []string) int {
	result := renvoUnaryNotStructMake()
	if result.value == 1 {
		print("PASS\n")
	}
	return 0
}
