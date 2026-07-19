package main

func appMain(args []string) int {
	fd := open("renvo_0739_open.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0739 open failed\n")
		return 1
	}
	b := []byte("x")
	if write(fd, b, 0) != 1 {
		print("RENVO-0739 write failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0739 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
