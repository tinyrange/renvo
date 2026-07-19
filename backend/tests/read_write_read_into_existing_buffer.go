package main

func appMain(args []string) int {
	fd := open("renvo_0760_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0760 open failed\n")
		return 1
	}
	b := []byte("abcd")
	if write(fd, b, 0) != 4 {
		print("RENVO-0760 seed write failed\n")
		return 1
	}
	r := []byte("zz")
	if read(fd, r, 1) != 2 {
		print("RENVO-0760 read failed\n")
		return 1
	}
	if r[0] != 'b' {
		print("RENVO-0760 value failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0760 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
