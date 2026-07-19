package main

func renvo0515All(s string, i int, want byte) bool {
	if i >= len(s) {
		return true
	}
	if s[i] != want {
		return false
	}
	return renvo0515All(s, i+1, want)
}

func appMain(args []string) int {
	var b byte = 'x'
	p := &b
	if !renvo0515All("xxx", 0, *p) {
		print("RENVO-0515 byte recursion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
