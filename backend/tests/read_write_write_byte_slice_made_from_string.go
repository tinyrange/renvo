package main

func appMain(args []string) int {
	fd := open("renvo_0753_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0753 open failed\n")
		return 1
	}
	b := []byte("go")
	if write(fd, b, 0) != 2 {
		print("RENVO-0753 write failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0753 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
