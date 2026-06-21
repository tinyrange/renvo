package main

type rtgSL6Score int

func appMain(args []string) int {
	scores := []rtgSL6Score{3, 5, 8}
	total := int(scores[0]) + int(scores[1]) + int(scores[2])
	if len(scores) != 3 || total != 16 {
		print("slice_literals_named_element_type failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
