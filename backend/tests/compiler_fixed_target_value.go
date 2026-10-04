package main

var renvoFixedTarget int = 0

func appMain() int {
	selected := 1
	if renvoFixedTarget != 0 {
		selected = renvoFixedTarget
	}
	if selected == 0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
