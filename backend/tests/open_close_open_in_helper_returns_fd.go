package main

func renvo0744Open() int {
	return open("renvo_0744_open.tmp", O_RDWR|O_CREATE|O_TRUNC)
}

func appMain(args []string) int {
	fd := renvo0744Open()
	if fd < 0 {
		print("RENVO-0744 helper open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0744 helper close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
