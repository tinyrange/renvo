package main

var groupedAddressValues [4]int

func groupedAddressIncrement(value *int) int {
	*value = *value + 1
	return *value
}

func appMain(args []string) int {
	index := 0
	for step := 0; step < 3; step++ {
		groupedAddressValues[index] = step + 1
		if groupedAddressIncrement(&index) >= len(groupedAddressValues) {
			break
		}
	}
	if groupedAddressValues[0] == 1 && groupedAddressValues[1] == 2 && groupedAddressValues[2] == 3 && groupedAddressValues[3] == 0 {
		print("PASS\n")
		return 0
	}
	return 1
}
