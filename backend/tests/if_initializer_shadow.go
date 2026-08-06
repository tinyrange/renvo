package main

func appMain(args []string) int {
	value := 7
	result := value
	if value := value + 5; value == 12 {
		result += value
	}
	if value != 7 || result != 19 {
		print("FAIL: if initializer shadowing\n")
		return 1
	}
	print("PASS\n")
	return 0
}
