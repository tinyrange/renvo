package main

var renvoRuntimeCheckGlobalInitializer = renvoRuntimeCheckInitializeGlobal()

func renvoRuntimeCheckInitializeGlobal() int {
	values := []byte("x")
	if values[0] == 'x' {
		return 1
	}
	return 0
}

func appMain() int {
	if renvoRuntimeCheckGlobalInitializer != 1 {
		return 1
	}
	print("PASS\n")
	return 0
}
