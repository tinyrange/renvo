package main

func appMain(args []string) int {
	x := 1
	goto done
	x = 9
done:
	if x != 1 {
		print("RENVO-0451 forward goto failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
