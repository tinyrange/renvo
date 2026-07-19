package main

func appMain(args []string) int {
	fd := open("renvo_0770_rw.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0770 open failed\n")
		return 1
	}
	b := []byte("abcd")
	if write(fd, b, 0) != 4 {
		print("RENVO-0770 seed write failed\n")
		return 1
	}
	r := []byte("  ")
	if read(fd, r, int64(1+1)) != 2 {
		print("RENVO-0770 offset expr read failed\n")
		return 1
	}
	if r[0] != 'c' {
		print("RENVO-0770 offset expr value failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0770 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
