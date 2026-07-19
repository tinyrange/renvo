package main

func appMain(args []string) int {
	fd := open("renvo_0777_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0777 open failed\n")
		return 1
	}
	if chmod(fd, 384) != 0 {
		print("RENVO-0777 chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0777 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
