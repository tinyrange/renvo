package main

func reusedScopeOffsetValue() string {
	{
		var expired *int
		_ = &expired
	}
	var current string
	current = "renvo"
	return current
}

func appMain(args []string) int {
	if reusedScopeOffsetValue() != "renvo" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
