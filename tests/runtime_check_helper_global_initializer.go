package main

var rtgRuntimeCheckGlobalInitializer = rtgRuntimeCheckInitializeGlobal()

func rtgRuntimeCheckInitializeGlobal() int {
	values := []byte("x")
	if values[0] == 'x' {
		return 1
	}
	return 0
}

func appMain() int {
	if rtgRuntimeCheckGlobalInitializer != 1 {
		return 1
	}
	print("PASS\n")
	return 0
}
