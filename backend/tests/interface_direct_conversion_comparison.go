package main

type directConversionValue struct {
	number int
}

func appMain(args []string) int {
	pointer := &directConversionValue{number: 7}
	var dynamic any = pointer
	if dynamic != any(pointer) {
		print("FAIL: direct interface conversion compared unequal\n")
		return 1
	}
	print("PASS\n")
	return 0
}
