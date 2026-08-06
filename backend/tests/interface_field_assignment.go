package main

type interfaceFieldAssignmentEntry struct {
	key interface{}
}

func interfaceFieldAssignmentEqual(key interface{}) bool {
	var entry interfaceFieldAssignmentEntry
	entry.key = key
	return entry.key == key
}

func appMain() int {
	if !interfaceFieldAssignmentEqual(7) || !interfaceFieldAssignmentEqual("seven") {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
