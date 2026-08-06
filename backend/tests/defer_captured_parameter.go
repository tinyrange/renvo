package main

func setDeferredCapturedParameter(out *int) {
	value := 1
	defer func() { *out = value }()
	value = 2
}

func appMain() int {
	got := 0
	setDeferredCapturedParameter(&got)
	if got != 2 {
		print("FAIL: captured parameter\n")
		return 1
	}
	print("PASS\n")
	return 0
}
