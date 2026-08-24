package main

var deferScopedReuseValue int

func deferScopedReuseRecord(value *int) {
	deferScopedReuseValue = *value
}

func runDeferScopedPointerSlotReuse() {
	deferScopedReuseValue = 0
	{
		value := 7
		defer deferScopedReuseRecord(&value)
	}
	overwrite := 99
	_ = &overwrite
}

func appMain(args []string) int {
	runDeferScopedPointerSlotReuse()
	if deferScopedReuseValue != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
