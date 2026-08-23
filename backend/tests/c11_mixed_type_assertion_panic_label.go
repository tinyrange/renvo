package main

// renvo:c11

func unusedC11MixedTypeAssertionPanicLabel() {}

func c11MixedTypeAssertionValue(value interface{}) int {
	return value.(int)
}

func appMain() int {
	var dynamic interface{} = 7
	if c11MixedTypeAssertionValue(dynamic) != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
