package main

func renvo0745Close() int {
	fd := open("renvo_0745_open.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		return 2
	}
	if close(fd) != 0 {
		return 3
	}
	return 0
}

func appMain(args []string) int {
	if renvo0745Close() != 0 {
		print("RENVO-0745 helper close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
