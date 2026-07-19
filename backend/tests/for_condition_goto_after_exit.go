package main

func appMain(args []string) int {
	i := 0
	for i < 3 {
		i = i + 1
	}
	goto check
	print("RENVO-0396 skipped failure\n")
	return 1
check:
	if i != 3 {
		print("RENVO-0396 goto after loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
