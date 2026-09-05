package main

func caughtNil(dynamic bool) (caught bool) {
	defer func() { caught = recover() != nil }()
	if dynamic {
		var value interface{}
		panic(value)
	}
	panic(nil)
	return
}

func appMain(args []string) int {
	if !caughtNil(false) || !caughtNil(true) {
		return 1
	}
	print("PASS\n")
	return 0
}
