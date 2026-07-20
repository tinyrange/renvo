package main

type renvoAppendOverlapValue struct {
	a int
	b int
}

func appMain(args []string) int {
	values := make([]renvoAppendOverlapValue, 5)
	values[0].a = 1
	values[1].a = 2
	values[2].a = 3
	values[3].a = 4
	result := append(values[:2], values[1:4]...)
	if len(result) != 5 || result[0].a != 1 || result[1].a != 2 || result[2].a != 2 || result[3].a != 3 || result[4].a != 4 {
		return 1
	}
	print("PASS\n")
	return 0
}
