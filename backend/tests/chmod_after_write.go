package main

func appMain(args []string) int {
	fd := open("renvo_0784_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0784 open failed\n")
		return 1
	}
	b := []byte("x")
	if write(fd, b, 0) != 1 {
		print("RENVO-0784 write failed\n")
		return 1
	}
	if chmod(fd, 420) != 0 {
		print("RENVO-0784 chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0784 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
