package main

func appMain(args []string) int {
	i := 0
	for {
		i++
		if i == 6 {
			break
		}
	}
	if i != 6 {
		print("RENVO-INCDEC-006 loop body increment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
