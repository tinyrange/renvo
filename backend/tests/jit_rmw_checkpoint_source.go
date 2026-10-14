package main

// Supplemental parsed-source semantics for the native RMW checkpoint bug.
// The actual failing native-emission reproducer is
// internal/rfe/runtime/native_loop_memory_retirement_test.go.
func appMain(args []string) int {
	values := make([]int, 2)
	values[0] = 10
	completed := 0
	for i := 0; i < 3; i++ {
		values[0] = values[0] + 1
		completed++
		if values[0] == 14 {
			break
		}
		values[0] = values[0] + 2
		completed++
	}
	if values[0] != 14 || completed != 3 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
