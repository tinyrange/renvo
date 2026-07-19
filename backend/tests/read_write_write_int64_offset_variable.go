package main

func appMain(args []string) int {
	fd := open("renvo_0769_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0769 open failed\n")
		return 1
	}
	b := []byte("k")
	off := int64(3)
	if write(fd, b, off) != 1 {
		print("RENVO-0769 int64 offset write failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0769 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
