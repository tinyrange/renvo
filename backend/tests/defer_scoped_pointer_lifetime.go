package main

var deferScopedTrace int

func deferScopedRecord(value *int) {
	deferScopedTrace = deferScopedTrace*10 + *value
}

func runDeferScopedPointerLifetime() {
	deferScopedTrace = 0
	{
		outer := 2
		defer deferScopedRecord(&outer)
		{
			inner := 3
			defer deferScopedRecord(&inner)
		}
	}
}

func appMain(args []string) int {
	runDeferScopedPointerLifetime()
	if deferScopedTrace != 32 {
		return 1
	}
	print("PASS\n")
	return 0
}
