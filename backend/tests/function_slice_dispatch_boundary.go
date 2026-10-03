package main

func sliceDispatchValue() int { return 3 }

func appMain(args []string) int {
	values := []func() int{sliceDispatchValue}
	if values[0]() != 3 {
		return 1
	}
	print("PASS\n")
	return 0
}
