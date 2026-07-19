package main

func appMain(args []string, env []string) int {
	var values []int
	p := &values
	*p = append(*p, 40)
	*p = append(*p, 2)
	if len(*p) == 2 {
		if (*p)[0] == 40 {
			if (*p)[1] == 2 {
				print("PASS\n")
				return 0
			}
		}
	}
	print("FAIL\n")
	return 1
}
