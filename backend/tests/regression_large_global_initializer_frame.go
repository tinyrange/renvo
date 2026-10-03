package main

// Initializing this value needs more than the 16-bit ENTER frame limit.
// Calls used to clear and copy it must not overwrite the unreserved tail.
var largeGlobalFrame = [9000]int{17}

func appMain() int {
	if largeGlobalFrame[0] != 17 {
		print("FAIL first element\n")
		return 1
	}
	for i := 1; i < len(largeGlobalFrame); i++ {
		if largeGlobalFrame[i] != 0 {
			print("FAIL initializer frame\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
