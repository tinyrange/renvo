package main

func appMain(args []string) int {
	x := 0
	goto first
second:
	x = x + 2
	goto done
first:
	x = x + 3
	goto second
done:
	if x != 5 {
		print("RENVO-0461 multiple labels failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
