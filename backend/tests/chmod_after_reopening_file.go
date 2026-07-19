package main

func appMain(args []string) int {
	fd := open("renvo_0792_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0792 open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0792 first close failed\n")
		return 1
	}
	fd = open("renvo_0792_chmod.tmp", O_RDWR)
	if fd < 0 {
		print("RENVO-0792 reopen failed\n")
		return 1
	}
	if chmod(fd, 420) != 0 {
		print("RENVO-0792 chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0792 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
