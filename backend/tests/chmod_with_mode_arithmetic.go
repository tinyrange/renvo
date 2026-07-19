package main

func appMain(args []string) int {
	fd := open("renvo_0789_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0789 open failed\n")
		return 1
	}
	if chmod(fd, 400+20) != 0 {
		print("RENVO-0789 chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0789 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
