package main

func appMain(args []string) int {
	fd := open("renvo_0798_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0798 open failed\n")
		return 1
	}
	b := []byte("w")
	if chmod(fd, 420) != 0 {
		print("RENVO-0798 chmod failed\n")
		return 1
	}
	if write(fd, b, 0) != 1 {
		print("RENVO-0798 write after chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0798 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
