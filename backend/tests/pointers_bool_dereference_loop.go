package main

func appMain(args []string) int {
	ok := false
	p := &ok
	count := 0
	for !*p {
		count = count + 1
		if count == 2 {
			*p = true
		}
	}
	if count != 2 {
		print("RENVO-0633 bool dereference loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
