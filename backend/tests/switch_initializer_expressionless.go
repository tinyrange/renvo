package main

func appMain(args []string) int {
	switch value := int64(0); {
	case value != 0:
		print("FAIL: wrong expressionless switch case\n")
		return 1
	default:
		print("PASS\n")
		return 0
	}
}
