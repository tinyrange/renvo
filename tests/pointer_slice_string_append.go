package main

func add(seen *[]string, value string) {
	values := *seen
	if contains(values, value) {
		return
	}
	values = append(values, value)
	*seen = values
}

func contains(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}

func appMain(args []string, env []string) int {
	var seen []string
	add(&seen, ".")
	add(&seen, "./rtg/cmd/rtg")
	if !contains(seen, ".") {
		print("FAIL\n")
		return 1
	}
	if !contains(seen, "./rtg/cmd/rtg") {
		print("FAIL\n")
		return 1
	}
	if contains(seen, "missing") {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
