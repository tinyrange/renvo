package main

func appMain(args []string) int {
	i := 0
	for i < 5 {
		if i == 2 {
			goto out
		}
		i = i + 1
	}
out:
	if i != 2 {
		print("RENVO-0456 goto out loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
