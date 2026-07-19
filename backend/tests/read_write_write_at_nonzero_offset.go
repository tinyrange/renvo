package main

func appMain(args []string) int {
	fd := open("renvo_0755_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0755 open failed\n")
		return 1
	}
	b := []byte("Z")
	if write(fd, b, 2) != 1 {
		print("RENVO-0755 offset write failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0755 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
