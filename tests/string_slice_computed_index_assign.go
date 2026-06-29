package main

func appMain() int {
	values := []string{"b", "a", ""}
	j := 1
	value := values[j]
	values[j+1] = values[j]
	values[j] = values[j-1]
	values[j-1] = value
	if values[0] != "a" {
		print("FAIL\n")
		return 1
	}
	if values[1] != "b" {
		print("FAIL\n")
		return 1
	}
	if values[2] != "a" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
