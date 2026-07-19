package main

func appMain(args []string) int {
	fd := open("renvo_0780_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0780 open failed\n")
		return 1
	}
	if chmod(fd, 0b110100100) != 0 {
		print("RENVO-0780 chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0780 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
