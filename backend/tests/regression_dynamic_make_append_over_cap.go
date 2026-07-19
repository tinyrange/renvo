package main

func dynamicCap(args []string) int {
	return len(args) + 4
}

func appMain(args []string) int {
	xs := make([]int, 0, dynamicCap(args))
	i := 0
	for i < 80 {
		xs = append(xs, i)
		i++
	}

	scratch := make([]byte, 0, len(args)+700)
	j := 0
	for j < 700 {
		scratch = append(scratch, byte('x'))
		j++
	}

	if len(xs) == 80 {
		if xs[4] == 4 {
			if xs[79] == 79 {
				if len(scratch) == 700 {
					print("PASS\n")
					return 0
				}
			}
		}
	}
	print("FAIL\n")
	return 1
}
