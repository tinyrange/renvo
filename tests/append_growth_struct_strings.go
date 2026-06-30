package main

type pair struct {
	left  string
	right string
}

func appMain(args []string) int {
	var values []pair
	i := 0
	for i < 80 {
		item := pair{left: "left", right: "right"}
		values = append(values, item)
		i++
	}
	if len(values) != 80 {
		print("FAIL\n")
		return 1
	}
	if values[0].left != "left" || values[0].right != "right" {
		print("FAIL\n")
		return 1
	}
	if values[79].left != "left" || values[79].right != "right" {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
