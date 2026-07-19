package main

func appMain(args []string) int {
	fd := open("renvo_0758_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0758 open failed\n")
		return 1
	}
	b := []byte("abcd")
	if write(fd, b, 0) != 4 {
		print("RENVO-0758 seed write failed\n")
		return 1
	}
	r := []byte("    ")
	if read(fd, r, 0) != 4 {
		print("RENVO-0758 read failed\n")
		return 1
	}
	if r[3] != 'd' {
		print("RENVO-0758 value failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0758 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
