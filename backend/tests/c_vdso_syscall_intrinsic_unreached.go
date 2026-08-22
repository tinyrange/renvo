package main

func renvo_runtime_CVDSOSyscall(number uint64, arg0 uint64, arg1 uint64, arg2 uint64) int64 {
	return 0
}

func appMain(args []string) int {
	if len(args) < 0 {
		_ = renvo_runtime_CVDSOSyscall(228, 1, 2, 3)
	}
	print("PASS\n")
	return 0
}
