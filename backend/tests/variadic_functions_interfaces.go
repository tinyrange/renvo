package main

func checkVariadicInterfaces(values ...interface{}) bool {
	if len(values) != 3 {
		return false
	}
	first := values[0]
	second := values[1]
	third := values[2]
	switch first.(type) {
	case string:
		if first.(string) != "value" {
			return false
		}
	default:
		return false
	}
	switch second.(type) {
	case int:
		if second.(int) != 42 {
			return false
		}
	default:
		return false
	}
	switch third.(type) {
	case bool:
		if !third.(bool) {
			return false
		}
	default:
		return false
	}
	return true
}

func appMain() int {
	if !checkVariadicInterfaces("value", 42, true) {
		return 1
	}
	print("PASS\n")
	return 0
}
