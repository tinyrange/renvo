package main

type interfaceNegativeInt64 int64

func interfaceNegativeInt64Value(value interface{}) int64 {
	number, ok := value.(interfaceNegativeInt64)
	if !ok {
		return 99
	}
	return int64(number)
}

func appMain(args []string) int {
	var value interface{} = interfaceNegativeInt64(-2)
	if interfaceNegativeInt64Value(value) != -2 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
