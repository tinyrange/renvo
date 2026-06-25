package main

func hasPrefix(s string, prefix string) bool {
	return s[0:len(prefix)] == prefix
}

func appMain() int {
	if hasPrefix("prefix-body", "prefix") {
		print("PASS\n")
		return 0
	}
	print("string_slice_compare_return failed\n")
	return 1
}
