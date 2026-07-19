package main

const targetAmd64 = 1

var currentTarget int = targetAmd64

func selectedValue() int {
	if currentTarget == targetAmd64 {
		return amd64Value()
	}
	if currentTarget == 2 {
		return 2
	}
	return 3
}

func amd64Value() int {
	return 1
}

func appMain(args []string) int {
	if selectedValue() != 1 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
