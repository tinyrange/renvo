package main

func appMain(args []string) int {
	fd := open("renvo_0741_open.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0741 open failed\n")
		return 1
	}
	if chmod(fd, 420) != 0 {
		print("RENVO-0741 chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0741 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
