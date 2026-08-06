package main

func deferredNamedResult(value int64) (result int64) {
	defer func() { result = result }()
	return value
}

func appMain(args []string) int {
	if deferredNamedResult(7) != 7 {
		print("FAIL: deferred closure saw a stale named result\n")
		return 1
	}
	print("PASS\n")
	return 0
}
