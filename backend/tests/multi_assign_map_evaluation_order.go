package main

func appMain() int {
	values := map[string]int{"old": 3}
	key := "old"
	key, values[key] = "new", 7
	if key != "new" || values["old"] != 7 || values["new"] != 0 {
		print("FAIL: multi-assignment map evaluation order\n")
		return 1
	}
	print("PASS\n")
	return 0
}
