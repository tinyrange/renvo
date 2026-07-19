package main

func choose(flag bool, a int, b int) int {
	if flag {
		return a * 3
	}
	return b + 4
}

func appMain(args []string) int {
	value := 0
	useFirst := true
	if useFirst {
		goto first
	}
	value = choose(false, 2, 5)
	goto done
first:
	value = choose(true, 4, 9)
done:
	if value != 12 {
		print("RENVO-0837 if selected call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
