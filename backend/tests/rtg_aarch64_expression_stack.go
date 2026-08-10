package main

func identity(value int) int {
	return value
}

func appMain(args []string) int {
	if identity(7)+identity(5) != 12 {
		return 1
	}
	print("PASS\n")
	return 0
}
