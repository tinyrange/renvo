package main

var renvo0688Wide int64 = 123456

func appMain(args []string) int {
	if int(renvo0688Wide) != 123456 {
		print("RENVO-0688 int64 global init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
