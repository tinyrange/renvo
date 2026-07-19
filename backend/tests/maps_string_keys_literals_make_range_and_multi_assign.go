package main

func appMain(args []string) int {
	key := "a"
	values := []int{1}
	first, second := map[string]int{key: values[0]}, map[string]int{"b": 2}
	var made map[string]int
	made = make(map[string]int)
	made["x"] = 3
	other := 4
	first["a"], other = other, first["a"]
	rangedKey := ""
	for current := range second {
		rangedKey = current
	}
	inline := "a"
	if first[inline] != 4 || other != 1 || second["b"] != 2 || made["x"] != 3 || rangedKey != "b" {
		print("maps_string_keys_literals_make_range_and_multi_assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
