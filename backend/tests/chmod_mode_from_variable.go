package main

func appMain(args []string) int {
	fd := open("renvo_0782_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0782 open failed\n")
		return 1
	}
	mode := 420
	if chmod(fd, mode) != 0 {
		print("RENVO-0782 chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0782 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
