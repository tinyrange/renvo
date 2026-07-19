package main

func typeAssertionValue(value interface{}) int {
	return value.(int)
}

func appMain() int {
	var dynamic interface{} = 7
	if typeAssertionValue(dynamic) != 7 {
		return 1
	}
	matched := false
	switch dynamic.(type) {
	case string:
		return 1
	case int:
		matched = true
	}
	if !matched {
		return 1
	}
	print("PASS\n")
	return 0
}
