package main

func appMain(args []string) int {
	fd := open("renvo_0765_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0765 open failed\n")
		return 1
	}
	var b []byte
	if write(fd, b, 0) != 0 {
		print("RENVO-0765 empty write failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0765 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
