package main

var stringArrayDynamicGlobal = [3]string{"zero", "one", "two"}

func stringArrayDynamicChoose(index int) string {
	return stringArrayDynamicGlobal[index]
}

func appMain() int {
	local := [3]string{"red", "green", "blue"}
	index := 1
	if stringArrayDynamicChoose(index) != "one" || local[index+1] != "blue" {
		print("FAIL\n")
		return 1
	}
	pointer := &local
	if pointer[index] != "green" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
